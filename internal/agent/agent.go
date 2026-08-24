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

// Package agent implements the ARTF gRPC service
package agent

import (
	"context"
	"log"
	"time"

	"github.com/iabtechlab/agentic-rtb-framework/internal/handlers"
	pb "github.com/iabtechlab/agentic-rtb-framework/pkg/pb/artf"
	"google.golang.org/grpc"
)

// ARTFAgent implements the RTBExtensionPoint gRPC service
type ARTFAgent struct {
	pb.UnimplementedRTBExtensionPointServer
	handlers *handlers.MutationHandlers
}

// NewARTFAgent creates a new ARTF agent instance
func NewARTFAgent(h *handlers.MutationHandlers) *ARTFAgent {
	return &ARTFAgent{
		handlers: h,
	}
}

// GetMutations processes an RTB request and returns proposed mutations.
// Respects applicable_intents from the request to filter which mutation types are returned.
func (a *ARTFAgent) GetMutations(ctx context.Context, req *pb.RTBRequest) (*pb.RTBResponse, error) {
	startTime := time.Now()

	// Check timeout budget
	tmax := req.GetTmax()
	if tmax > 0 {
		deadline := time.Now().Add(time.Duration(tmax) * time.Millisecond)
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, deadline)
		defer cancel()
	}

	// Collect mutations from all registered handlers
	var mutations []*pb.Mutation

	// Process based on lifecycle stage
	lifecycle := req.GetLifecycle()
	bidRequest := req.GetBidRequest()
	bidResponse := req.GetBidResponse()

	applicableIntents := req.GetApplicableIntents()

	log.Printf("Processing request %s at lifecycle stage %v with applicable_intents=%v",
		req.GetId(), lifecycle, applicableIntents)

	// Run segment activation handler
	if segmentMutations, err := a.handlers.ProcessSegments(ctx, bidRequest, applicableIntents); err == nil {
		mutations = append(mutations, segmentMutations...)
	} else {
		log.Printf("Segment processing error: %v", err)
	}

	// Run deal activation handler
	if dealMutations, err := a.handlers.ProcessDeals(ctx, bidRequest, applicableIntents); err == nil {
		mutations = append(mutations, dealMutations...)
	} else {
		log.Printf("Deal processing error: %v", err)
	}

	// Run bid shading handler (if bid response is present)
	if bidResponse != nil {
		if bidMutations, err := a.handlers.ProcessBidShading(ctx, bidRequest, bidResponse, applicableIntents); err == nil {
			mutations = append(mutations, bidMutations...)
		} else {
			log.Printf("Bid shading error: %v", err)
		}
	}

	// Run content data handler for ADD_CIDS
	if contentMutations, err := a.handlers.ProcessContentData(ctx, bidRequest, applicableIntents); err == nil {
		mutations = append(mutations, contentMutations...)
	} else {
		log.Printf("Content data processing error: %v", err)
	}

	// Build response
	response := &pb.RTBResponse{
		Id:        req.Id,
		Mutations: mutations,
		Metadata: &pb.Metadata{
			ApiVersion:   stringPtr("1.0"),
			ModelVersion: stringPtr("v0.10.0"),
		},
	}

	log.Printf("Request %s processed in %v, returning %d mutations",
		req.GetId(), time.Since(startTime), len(mutations))

	return response, nil
}

// LoggingInterceptor logs gRPC requests
func LoggingInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	log.Printf("gRPC %s took %v, error: %v", info.FullMethod, time.Since(start), err)
	return resp, err
}

// RegisterRTBExtensionPointServer registers the service with a gRPC server
func RegisterRTBExtensionPointServer(s *grpc.Server, agent *ARTFAgent) {
	pb.RegisterRTBExtensionPointServer(s, agent)
}

func stringPtr(s string) *string {
	return &s
}
