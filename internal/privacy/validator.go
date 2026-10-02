package privacy

import (
	"errors"
	"time"

	pb "github.com/IABTechLab/agentic-rtb-framework/pkg/pb/artf"
	privacy_pb "github.com/IABTechLab/agentic-rtb-framework/pkg/pb/privacy"
)

var (
	ErrConsentDenied       = errors.New("privacy: consent denied")
	ErrConsentUnknown      = errors.New("privacy: consent status unknown")
	ErrPIINotAllowed       = errors.New("privacy: personal data not allowed")
	ErrRetentionExceeded   = errors.New("privacy: data retention limit exceeded")
	ErrGeoRestriction      = errors.New("privacy: geographic restriction violated")
	ErrUnauthorizedAgent   = errors.New("privacy: agent not authorized")
	ErrProhibitedEnrichment = errors.New("privacy: enrichment type prohibited")
)

// Validator validates privacy context and enforces constraints
type Validator struct {
	agentID       string
	strictMode    bool
	telemetryFunc func(event string, metadata map[string]interface{})
}

// NewValidator creates a new privacy validator
func NewValidator(agentID string, strictMode bool) *Validator {
	return &Validator{
		agentID:    agentID,
		strictMode: strictMode,
	}
}

// SetTelemetry configures optional telemetry callback
func (v *Validator) SetTelemetry(f func(event string, metadata map[string]interface{})) {
	v.telemetryFunc = f
}

// Validate enforces privacy constraints on the request
func (v *Validator) Validate(ctx *privacy_pb.PrivacyContext) (*privacy_pb.PrivacyValidationPayload, error) {
	if ctx == nil {
		return &privacy_pb.PrivacyValidationPayload{
			Result: &privacy_pb.ValidationResult{
				Compliant:       true,
				ComplianceScore: 1.0,
				Reason:          "no privacy context provided",
			},
			Action: &privacy_pb.EnforcementAction{
				Action: privacy_pb.EnforcementAction_ALLOW,
			},
		}, nil
	}

	v.emitTelemetry("privacy.context.received", map[string]interface{}{
		"agent_id": v.agentID,
	})

	violations := make([]*privacy_pb.PolicyViolation, 0)

	// Validate consent
	if err := v.validateConsent(ctx.Consent); err != nil {
		violations = append(violations, &privacy_pb.PolicyViolation{
			Type:        privacy_pb.PolicyViolation_CONSENT_DENIED,
			Description: err.Error(),
			Severity:    10,
		})
	}

	// Validate agent authorization
	if err := v.validateAgentAuthorization(ctx.ProcessingConstraints); err != nil {
		violations = append(violations, &privacy_pb.PolicyViolation{
			Type:        privacy_pb.PolicyViolation_UNAUTHORIZED_AGENT,
			Description: err.Error(),
			Severity:    9,
		})
	}

	// Validate data controls
	action := v.determineEnforcementAction(ctx.DataControls, violations)

	compliant := len(violations) == 0
	score := v.calculateComplianceScore(violations)

	result := &privacy_pb.PrivacyValidationPayload{
		Result: &privacy_pb.ValidationResult{
			Compliant:              compliant,
			ComplianceScore:        score,
			ValidationTimestampMs:  time.Now().UnixMilli(),
			Reason:                 v.buildReasonString(compliant, violations),
		},
		Violations: violations,
		Action:     action,
	}

	if !compliant {
		v.emitTelemetry("privacy.policy.violation", map[string]interface{}{
			"agent_id":        v.agentID,
			"violation_count": len(violations),
			"action":          action.Action.String(),
		})
	} else {
		v.emitTelemetry("privacy.consent.validated", map[string]interface{}{
			"agent_id": v.agentID,
			"score":    score,
		})
	}

	return result, nil
}

// validateConsent checks consent status
func (v *Validator) validateConsent(consent *privacy_pb.Consent) error {
	if consent == nil {
		if v.strictMode {
			return ErrConsentUnknown
		}
		return nil
	}

	switch consent.Status {
	case privacy_pb.Consent_DENIED:
		return ErrConsentDenied
	case privacy_pb.Consent_UNKNOWN:
		if v.strictMode {
			return ErrConsentUnknown
		}
	case privacy_pb.Consent_GRANTED, privacy_pb.Consent_NOT_REQUIRED:
		// Check consent expiration
		if consent.TimestampMs > 0 {
			age := time.Now().UnixMilli() - consent.TimestampMs
			// Consent expires after 13 months (GDPR)
			if age > 13*30*24*60*60*1000 {
				return errors.New("consent expired")
			}
		}
	}

	return nil
}

// validateAgentAuthorization checks if this agent is allowed to process
func (v *Validator) validateAgentAuthorization(constraints *privacy_pb.ProcessingConstraints) error {
	if constraints == nil || len(constraints.AllowedAgents) == 0 {
		return nil
	}

	for _, allowed := range constraints.AllowedAgents {
		if allowed == v.agentID {
			return nil
		}
	}

	return ErrUnauthorizedAgent
}

// determineEnforcementAction decides what action to take
func (v *Validator) determineEnforcementAction(controls *privacy_pb.DataControls, violations []*privacy_pb.PolicyViolation) *privacy_pb.EnforcementAction {
	// If critical violations exist, block
	for _, violation := range violations {
		if violation.Severity >= 9 {
			return &privacy_pb.EnforcementAction{
				Action:      privacy_pb.EnforcementAction_BLOCK,
				Explanation: "critical privacy violation detected",
			}
		}
	}

	// If PII not allowed, strip it
	if controls != nil && controls.PersonalDataAllowed != nil && !*controls.PersonalDataAllowed {
		v.emitTelemetry("privacy.pii.stripped", map[string]interface{}{
			"agent_id": v.agentID,
		})
		return &privacy_pb.EnforcementAction{
			Action:         privacy_pb.EnforcementAction_STRIP_PII,
			AffectedFields: []string{"user.id", "user.buyeruid", "device.ifa", "device.dpidmd5"},
			Explanation:    "PII removed due to data controls",
		}
	}

	// If anonymization required
	if controls != nil && controls.AnonymizationRequired != nil && *controls.AnonymizationRequired {
		return &privacy_pb.EnforcementAction{
			Action:      privacy_pb.EnforcementAction_ANONYMIZE,
			Explanation: "data anonymized per privacy context",
		}
	}

	// If moderate violations, downgrade to contextual
	if len(violations) > 0 {
		return &privacy_pb.EnforcementAction{
			Action:      privacy_pb.EnforcementAction_CONTEXTUAL_ONLY,
			Explanation: "downgraded to contextual auction due to privacy constraints",
		}
	}

	return &privacy_pb.EnforcementAction{
		Action:      privacy_pb.EnforcementAction_ALLOW,
		Explanation: "request compliant with privacy context",
	}
}

// calculateComplianceScore computes a score from 0.0 to 1.0
func (v *Validator) calculateComplianceScore(violations []*privacy_pb.PolicyViolation) float64 {
	if len(violations) == 0 {
		return 1.0
	}

	totalSeverity := 0
	for _, violation := range violations {
		totalSeverity += int(violation.Severity)
	}

	// Normalize to 0-1 range (assuming max severity 10 per violation)
	maxPossibleSeverity := len(violations) * 10
	score := 1.0 - (float64(totalSeverity) / float64(maxPossibleSeverity))

	if score < 0 {
		return 0
	}
	return score
}

// buildReasonString creates human-readable reason
func (v *Validator) buildReasonString(compliant bool, violations []*privacy_pb.PolicyViolation) string {
	if compliant {
		return "request compliant with privacy context"
	}

	if len(violations) == 1 {
		return violations[0].Description
	}

	return "multiple privacy violations detected"
}

// emitTelemetry sends telemetry event if configured
func (v *Validator) emitTelemetry(event string, metadata map[string]interface{}) {
	if v.telemetryFunc != nil {
		v.telemetryFunc(event, metadata)
	}
}

// EnforceRetention checks data retention limits
func EnforceRetention(ctx *privacy_pb.PrivacyContext, dataTimestampMs int64) error {
	if ctx == nil || ctx.DataControls == nil || ctx.DataControls.RetentionTtlMs == nil {
		return nil
	}

	age := time.Now().UnixMilli() - dataTimestampMs
	if age > *ctx.DataControls.RetentionTtlMs {
		return ErrRetentionExceeded
	}

	return nil
}

// StripPII removes personally identifiable information from bid request
func StripPII(bidRequest *pb.BidRequest) {
	if bidRequest.User != nil {
		bidRequest.User.Id = ""
		bidRequest.User.Buyeruid = ""
		bidRequest.User.Yob = 0
		bidRequest.User.Gender = ""
	}

	if bidRequest.Device != nil {
		bidRequest.Device.Ifa = ""
		bidRequest.Device.Dpidmd5 = ""
		bidRequest.Device.Dpidsha1 = ""
		bidRequest.Device.Macsha1 = ""
		bidRequest.Device.Macmd5 = ""
	}
}
