package handlers

import (
	"context"
	"fmt"

	pb "github.com/IABTechLab/agentic-rtb-framework/pkg/pb/artf"
	"github.com/IABTechLab/agentic-rtb-framework/internal/privacy"
)

// PrivacyAwareHandler wraps mutation handlers with privacy validation
type PrivacyAwareHandler struct {
	validator      *privacy.Validator
	baseHandler    MutationHandler
	telemetryFunc  func(event string, metadata map[string]interface{})
}

// MutationHandler is the base interface for processing mutations
type MutationHandler interface {
	GetMutations(ctx context.Context, req *pb.RTBRequest) (*pb.RTBResponse, error)
}

// NewPrivacyAwareHandler creates a handler with privacy enforcement
func NewPrivacyAwareHandler(agentID string, strictMode bool, baseHandler MutationHandler) *PrivacyAwareHandler {
	return &PrivacyAwareHandler{
		validator:   privacy.NewValidator(agentID, strictMode),
		baseHandler: baseHandler,
	}
}

// SetTelemetry configures telemetry callback
func (h *PrivacyAwareHandler) SetTelemetry(f func(event string, metadata map[string]interface{})) {
	h.telemetryFunc = f
	h.validator.SetTelemetry(f)
}

// GetMutations processes request with privacy enforcement
func (h *PrivacyAwareHandler) GetMutations(ctx context.Context, req *pb.RTBRequest) (*pb.RTBResponse, error) {
	// Extract privacy context from request extension
	privacyCtx := h.extractPrivacyContext(req)

	// Validate privacy constraints
	validationResult, err := h.validator.Validate(privacyCtx)
	if err != nil {
		return nil, fmt.Errorf("privacy validation failed: %w", err)
	}

	// Check enforcement action
	switch validationResult.Action.Action {
	case privacy_pb.EnforcementAction_BLOCK:
		// Return empty response with validation mutation
		return &pb.RTBResponse{
			Id: req.Id,
			Mutations: []*pb.Mutation{
				{
					Intent: pb.Intent_VALIDATE_PRIVACY,
					Op:     pb.Operation_OPERATION_ADD,
					Path:   "/privacy/validation",
					Value: &pb.Mutation_PrivacyValidation{
						PrivacyValidation: validationResult,
					},
				},
			},
			Metadata: &pb.Metadata{
				ApiVersion: "1.0",
			},
		}, nil

	case privacy_pb.EnforcementAction_STRIP_PII:
		// Strip PII from request before processing
		h.emitTelemetry("privacy.pii.stripped", map[string]interface{}{
			"request_id":      req.Id,
			"affected_fields": validationResult.Action.AffectedFields,
		})
		privacy.StripPII(req.BidRequest)

	case privacy_pb.EnforcementAction_ANONYMIZE:
		h.emitTelemetry("privacy.data.anonymized", map[string]interface{}{
			"request_id": req.Id,
		})
		// Apply anonymization (implementation depends on use case)

	case privacy_pb.EnforcementAction_CONTEXTUAL_ONLY:
		h.emitTelemetry("privacy.downgraded.contextual", map[string]interface{}{
			"request_id": req.Id,
		})
		// Remove user-level signals, keep only contextual data
		if req.BidRequest.User != nil {
			req.BidRequest.User = nil
		}
	}

	// Call base handler to process mutations
	response, err := h.baseHandler.GetMutations(ctx, req)
	if err != nil {
		return nil, err
	}

	// Add privacy validation mutation to response
	if !validationResult.Result.Compliant || len(validationResult.Violations) > 0 {
		response.Mutations = append([]*pb.Mutation{
			{
				Intent: pb.Intent_VALIDATE_PRIVACY,
				Op:     pb.Operation_OPERATION_ADD,
				Path:   "/privacy/validation",
				Value: &pb.Mutation_PrivacyValidation{
					PrivacyValidation: validationResult,
				},
			},
		}, response.Mutations...)
	}

	return response, nil
}

// extractPrivacyContext retrieves privacy context from request
func (h *PrivacyAwareHandler) extractPrivacyContext(req *pb.RTBRequest) *privacy_pb.PrivacyContext {
	if req.Ext == nil {
		return nil
	}
	return req.Ext.PrivacyContext
}

// emitTelemetry sends telemetry event if configured
func (h *PrivacyAwareHandler) emitTelemetry(event string, metadata map[string]interface{}) {
	if h.telemetryFunc != nil {
		h.telemetryFunc(event, metadata)
	}
}

// Example: SegmentActivationHandler with privacy awareness
type SegmentActivationHandler struct {
	agentID string
}

func NewSegmentActivationHandler(agentID string) *SegmentActivationHandler {
	return &SegmentActivationHandler{agentID: agentID}
}

func (h *SegmentActivationHandler) GetMutations(ctx context.Context, req *pb.RTBRequest) (*pb.RTBResponse, error) {
	// Example: activate segments based on user demographics
	mutations := make([]*pb.Mutation, 0)

	if req.BidRequest.User != nil && req.BidRequest.User.Yob > 0 {
		age := 2026 - int(req.BidRequest.User.Yob)
		
		var segments []string
		if age >= 18 && age <= 24 {
			segments = append(segments, "demo-18-24")
		} else if age >= 25 && age <= 34 {
			segments = append(segments, "demo-25-34")
		}

		if len(segments) > 0 {
			mutations = append(mutations, &pb.Mutation{
				Intent: pb.Intent_ACTIVATE_SEGMENTS,
				Op:     pb.Operation_OPERATION_ADD,
				Path:   "/user/data/segment",
				Value: &pb.Mutation_Ids{
					Ids: &pb.IDsPayload{
						Id: segments,
					},
				},
			})
		}
	}

	return &pb.RTBResponse{
		Id:        req.Id,
		Mutations: mutations,
		Metadata: &pb.Metadata{
			ApiVersion:   "1.0",
			ModelVersion: "segment-v1",
		},
	}, nil
}

// Example usage in main agent
func ExamplePrivacyAwareAgent() {
	// Create base handler
	baseHandler := NewSegmentActivationHandler("audience-agent")

	// Wrap with privacy enforcement
	privacyHandler := NewPrivacyAwareHandler("audience-agent", true, baseHandler)

	// Configure telemetry
	privacyHandler.SetTelemetry(func(event string, metadata map[string]interface{}) {
		fmt.Printf("Telemetry: %s - %v\n", event, metadata)
	})

	// Process request (in actual implementation, this would be called by gRPC handler)
	// ctx := context.Background()
	// response, err := privacyHandler.GetMutations(ctx, request)
}
