// Copyright (c) 2025 Index Exchange Inc.
//
// This file is part of the Agentic RTB Framework reference implementation.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

// Package mcp implements the MCP interface for ARTF.
// This package provides an MCP (Model Context Protocol) wrapper around the gRPC
// RTBExtensionPoint service, allowing AI agents to interact with ARTF via MCP tools.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/iabtechlab/agentic-rtb-framework/internal/agent"
	"github.com/iabtechlab/agentic-rtb-framework/internal/federation"
	pb "github.com/iabtechlab/agentic-rtb-framework/pkg/pb/artf"
	openrtb "github.com/iabtechlab/agentic-rtb-framework/pkg/pb/openrtb"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"google.golang.org/protobuf/encoding/protojson"
)

// Agent wraps the MCP server with ARTF-specific functionality.
// It delegates all business logic to the underlying gRPC ARTFAgent.
type Agent struct {
	mcpServer         *server.MCPServer
	grpcAgent         *agent.ARTFAgent
	addr              string
	port              int
	federationManager *federation.Manager
}

// SetFederationManager sets the federation manager for federated GRPC calls
func (a *Agent) SetFederationManager(fm *federation.Manager) {
	a.federationManager = fm
}

// GetFederationManager returns the federation manager
func (a *Agent) GetFederationManager() *federation.Manager {
	return a.federationManager
}

// NewAgent creates a new MCP agent instance that wraps the gRPC agent
func NewAgent(grpcAgent *agent.ARTFAgent, addr string, port int) *Agent {
	// Create MCP server
	mcpServer := server.NewMCPServer(
		"ARTF Agent",
		"0.10.0",
		server.WithToolCapabilities(true),
		server.WithRecovery(),
	)

	a := &Agent{
		mcpServer:  mcpServer,
		grpcAgent:  grpcAgent,
		addr:       addr,
		port:       port,
	}

	// Register the extend_rtb tool
	a.registerTools()

	return a
}

// registerTools registers the ARTF tools with the MCP server
func (a *Agent) registerTools() {
	// Define the extend_rtb tool with full ARTF protocol support
	extendRTBTool := mcp.NewTool("extend_rtb",
		mcp.WithDescription("Process an OpenRTB bid request/response and return proposed mutations. "+
			"Supports intents: ACTIVATE_SEGMENTS, ACTIVATE_DEALS, SUPPRESS_DEALS, ADJUST_DEAL_FLOOR, "+
			"ADJUST_DEAL_MARGIN, BID_SHADE, ADD_METRICS, ADD_CIDS. "+
			"Use applicable_intents to filter which mutation types you want returned."),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Unique request ID assigned by the exchange"),
		),
		mcp.WithNumber("tmax",
			mcp.Description("Maximum response time in milliseconds the exchange allows for mutations"),
		),
		mcp.WithObject("bid_request",
			mcp.Required(),
			mcp.Description("OpenRTB v2.6 BidRequest object"),
		),
		mcp.WithObject("bid_response",
			mcp.Description("OpenRTB v2.6 BidResponse object (required for BID_SHADE intent)"),
		),
		mcp.WithString("lifecycle",
			mcp.Description("Auction lifecycle stage: LIFECYCLE_PUBLISHER_BID_REQUEST or LIFECYCLE_DSP_BID_RESPONSE"),
		),
		mcp.WithObject("originator",
			mcp.Description("Business entity that created the BidRequest/BidResponse. Object with 'type' (TYPE_PUBLISHER, TYPE_SSP, TYPE_EXCHANGE, TYPE_DSP) and 'id' fields"),
		),
		mcp.WithArray("applicable_intents",
			mcp.Description("List of intents the agent is eligible to return. If omitted, all intents are applicable. "+
				"Valid values: ACTIVATE_SEGMENTS, ACTIVATE_DEALS, SUPPRESS_DEALS, ADJUST_DEAL_FLOOR, "+
				"ADJUST_DEAL_MARGIN, BID_SHADE, ADD_METRICS, ADD_CIDS"),
		),
		mcp.WithBoolean("federate",
			mcp.Description("If true, also call configured federated GRPC endpoints and aggregate their mutations. Default: false."),
		),
		mcp.WithArray("federate_endpoints",
			mcp.Description("List of specific endpoint names to call. If empty and federate=true, all applicable endpoints are called."),
		),
	)

	// Define federation tools
	listFederatedEndpointsTool := mcp.NewTool("list_federated_endpoints",
		mcp.WithDescription("List all configured federated GRPC endpoints, their acceptable intents, health status, and configuration. Returns empty list if federation is not configured."),
	)

	// Register tools with handlers
	a.mcpServer.AddTool(extendRTBTool, a.handleExtendRTB)
	a.mcpServer.AddTool(listFederatedEndpointsTool, a.handleListFederatedEndpoints)
}

// handleExtendRTB processes the extend_rtb tool call by delegating to the gRPC agent
func (a *Agent) handleExtendRTB(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	startTime := time.Now()

	// Extract parameters
	id, err := request.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter: id"), nil
	}

	// Get optional tmax
	tmax := int32(100) // default
	if tmaxVal, err := request.RequireFloat("tmax"); err == nil {
		tmax = int32(tmaxVal)
	}

	// Get bid_request
	args := request.GetArguments()
	bidRequestRaw, ok := args["bid_request"].(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("missing required parameter: bid_request"), nil
	}

	// Get optional bid_response
	var bidResponseRaw map[string]interface{}
	if br, ok := args["bid_response"].(map[string]interface{}); ok {
		bidResponseRaw = br
	}

	// Get optional lifecycle
	var lifecycleStr string
	if lc, ok := args["lifecycle"].(string); ok && lc != "" {
		lifecycleStr = lc
	}
	lifecycle := parseLifecycle(lifecycleStr)

	// Get optional originator
	var originator *pb.Originator
	var originatorStr string
	if orig, ok := args["originator"].(map[string]interface{}); ok {
		originator = &pb.Originator{}
		if t, ok := orig["type"].(string); ok {
			originatorStr = t
			originatorType := parseOriginatorType(t)
			originator.Type = originatorType.Enum()
		}
		if id, ok := orig["id"].(string); ok {
			originator.Id = &id
		}
	}

	// Get optional applicable_intents
	var applicableIntentStrs []string
	var applicableIntents []pb.Intent
	if intentsRaw, ok := args["applicable_intents"].([]interface{}); ok {
		for _, intentRaw := range intentsRaw {
			if intentStr, ok := intentRaw.(string); ok {
				applicableIntentStrs = append(applicableIntentStrs, intentStr)
				if parsed := parseIntent(intentStr); parsed != pb.Intent_INTENT_UNSPECIFIED {
					applicableIntents = append(applicableIntents, parsed)
				}
			}
		}
	}

	log.Printf("MCP: Processing extend_rtb request %s with tmax=%d, lifecycle=%s, originator=%s, applicable_intents=%v",
		id, tmax, lifecycleStr, originatorStr, applicableIntentStrs)

	// Convert JSON to protobuf
	bidRequest, err := jsonToOpenRTBBidRequest(bidRequestRaw)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to parse bid_request: %v", err)), nil
	}

	var bidResponse *openrtb.BidResponse
	if bidResponseRaw != nil {
		bidResponse, err = jsonToOpenRTBBidResponse(bidResponseRaw)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to parse bid_response: %v", err)), nil
		}
	}

	// Build the gRPC request
	grpcRequest := &pb.RTBRequest{
		Id:                &id,
		Tmax:              &tmax,
		Lifecycle:         lifecycle.Enum(),
		BidRequest:        bidRequest,
		BidResponse:       bidResponse,
		Originator:        originator,
		ApplicableIntents: applicableIntents,
	}

	// Call the gRPC agent directly (no network hop)
	grpcResponse, err := a.grpcAgent.GetMutations(ctx, grpcRequest)
	if err != nil {
		log.Printf("MCP: Error from gRPC agent: %v", err)
		return mcp.NewToolResultError(fmt.Sprintf("processing error: %v", err)), nil
	}

	// Check if federation is requested
	shouldFederate := false
	if fedVal, ok := args["federate"].(bool); ok {
		shouldFederate = fedVal
	}

	// If federation is enabled, call federated endpoints and merge results
	if shouldFederate && a.federationManager != nil {
		// Extract specific endpoint names if provided
		var endpointNames []string
		if epRaw, ok := args["federate_endpoints"].([]interface{}); ok {
			for _, ep := range epRaw {
				if name, ok := ep.(string); ok {
					endpointNames = append(endpointNames, name)
				}
			}
		}

		log.Printf("MCP: Federating request %s to remote endpoints (specific=%v)", id, endpointNames)

		var fedResponse *federation.FederatedResponse
		if len(endpointNames) > 0 {
			// Call specific endpoints
			fedResponse = &federation.FederatedResponse{ID: id}
			for _, epName := range endpointNames {
				resp, err := a.federationManager.CallEndpoint(ctx, epName, grpcRequest)
				if err != nil {
					log.Printf("MCP: Federation endpoint '%s' error: %v", epName, err)
					fedResponse.EndpointResults = append(fedResponse.EndpointResults, federation.FederatedResult{
						EndpointName: epName,
						Success:      false,
						Error:        err.Error(),
					})
				} else {
					fedResponse.Mutations = append(fedResponse.Mutations, resp.GetMutations()...)
					fedResponse.EndpointResults = append(fedResponse.EndpointResults, federation.FederatedResult{
						EndpointName: epName,
						Success:      true,
						Mutations:    resp.GetMutations(),
					})
				}
			}
		} else {
			// Call all applicable endpoints
			fedResponse, err = a.federationManager.GetMutations(ctx, grpcRequest, applicableIntentStrs)
			if err != nil {
				log.Printf("MCP: Federation error: %v", err)
			}
		}

		// Merge federated mutations with local mutations
		if fedResponse != nil && len(fedResponse.Mutations) > 0 {
			allMutations := grpcResponse.GetMutations()
			allMutations = append(allMutations, fedResponse.Mutations...)
			grpcResponse.Mutations = allMutations
			log.Printf("MCP: Merged %d federated mutations with %d local mutations",
				len(fedResponse.Mutations), len(grpcResponse.GetMutations())-len(fedResponse.Mutations))
		}
	}

	// Convert protobuf response to JSON
	jsonResponse, err := protoResponseToJSON(grpcResponse)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("serialization error: %v", err)), nil
	}

	log.Printf("MCP: Request %s processed in %v, returning %d mutations",
		id, time.Since(startTime), len(grpcResponse.GetMutations()))

	return mcp.NewToolResultText(string(jsonResponse)), nil
}

// handleListFederatedEndpoints lists configured federation endpoints
func (a *Agent) handleListFederatedEndpoints(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if a.federationManager == nil {
		result := map[string]interface{}{
			"endpoints": []interface{}{},
			"message":   "Federation is not configured. Use --federation-config flag to enable.",
		}
		responseJSON, _ := json.Marshal(result)
		return mcp.NewToolResultText(string(responseJSON)), nil
	}

	endpoints := a.federationManager.ListEndpoints()
	result := map[string]interface{}{
		"endpoints": endpoints,
		"count":     len(endpoints),
	}

	responseJSON, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("serialization error: %v", err)), nil
	}

	return mcp.NewToolResultText(string(responseJSON)), nil
}

// parseIntent converts a string to pb.Intent
func parseIntent(s string) pb.Intent {
	switch s {
	case "ACTIVATE_SEGMENTS":
		return pb.Intent_ACTIVATE_SEGMENTS
	case "ACTIVATE_DEALS":
		return pb.Intent_ACTIVATE_DEALS
	case "SUPPRESS_DEALS":
		return pb.Intent_SUPPRESS_DEALS
	case "ADJUST_DEAL_FLOOR":
		return pb.Intent_ADJUST_DEAL_FLOOR
	case "ADJUST_DEAL_MARGIN":
		return pb.Intent_ADJUST_DEAL_MARGIN
	case "BID_SHADE":
		return pb.Intent_BID_SHADE
	case "ADD_METRICS":
		return pb.Intent_ADD_METRICS
	case "ADD_CIDS":
		return pb.Intent_ADD_CIDS
	default:
		return pb.Intent_INTENT_UNSPECIFIED
	}
}

// parseLifecycle converts a string to pb.Lifecycle
func parseLifecycle(s string) pb.Lifecycle {
	switch s {
	case "LIFECYCLE_PUBLISHER_BID_REQUEST":
		return pb.Lifecycle_LIFECYCLE_PUBLISHER_BID_REQUEST
	case "LIFECYCLE_DSP_BID_RESPONSE":
		return pb.Lifecycle_LIFECYCLE_DSP_BID_RESPONSE
	default:
		return pb.Lifecycle_LIFECYCLE_UNSPECIFIED
	}
}

// parseOriginatorType converts a string to pb.Originator_Type
func parseOriginatorType(s string) pb.Originator_Type {
	switch s {
	case "TYPE_PUBLISHER":
		return pb.Originator_TYPE_PUBLISHER
	case "TYPE_SSP":
		return pb.Originator_TYPE_SSP
	case "TYPE_EXCHANGE":
		return pb.Originator_TYPE_EXCHANGE
	case "TYPE_DSP":
		return pb.Originator_TYPE_DSP
	default:
		return pb.Originator_TYPE_UNSPECIFIED
	}
}

// jsonToOpenRTBBidRequest converts JSON map to OpenRTB BidRequest protobuf
func jsonToOpenRTBBidRequest(data map[string]interface{}) (*openrtb.BidRequest, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	req := &openrtb.BidRequest{}
	if err := protojson.Unmarshal(jsonBytes, req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to protobuf: %w", err)
	}

	return req, nil
}

// jsonToOpenRTBBidResponse converts JSON map to OpenRTB BidResponse protobuf
func jsonToOpenRTBBidResponse(data map[string]interface{}) (*openrtb.BidResponse, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	resp := &openrtb.BidResponse{}
	if err := protojson.Unmarshal(jsonBytes, resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to protobuf: %w", err)
	}

	return resp, nil
}

// protoResponseToJSON converts the gRPC response to JSON for MCP
func protoResponseToJSON(resp *pb.RTBResponse) ([]byte, error) {
	// Use protojson for proper JSON serialization
	opts := protojson.MarshalOptions{
		UseProtoNames:   true,
		EmitUnpopulated: false,
	}
	return opts.Marshal(resp)
}

// corsMiddleware wraps an http.Handler with CORS headers
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Mcp-Session-Id, Last-Event-ID")
		w.Header().Set("Access-Control-Expose-Headers", "Mcp-Session-Id")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Start starts the MCP interface using Streamable HTTP transport
func (a *Agent) Start() error {
	listenAddr := fmt.Sprintf("%s:%d", a.addr, a.port)
	log.Printf("MCP interface starting on %s", listenAddr)

	// Create streamable HTTP server
	streamableServer := server.NewStreamableHTTPServer(a.mcpServer)

	// Create HTTP mux and wrap with CORS middleware
	mux := http.NewServeMux()
	mux.Handle("/mcp", corsMiddleware(streamableServer))

	// Create and start HTTP server
	httpServer := &http.Server{
		Addr:         listenAddr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return httpServer.ListenAndServe()
}

// GetMCPServer returns the underlying MCP server for custom configuration
func (a *Agent) GetMCPServer() *server.MCPServer {
	return a.mcpServer
}

// Handler returns an HTTP handler for the MCP endpoint with CORS support.
// This can be mounted on an existing mux to serve MCP alongside other routes.
func (a *Agent) Handler() http.Handler {
	streamableServer := server.NewStreamableHTTPServer(a.mcpServer)
	return corsMiddleware(streamableServer)
}
