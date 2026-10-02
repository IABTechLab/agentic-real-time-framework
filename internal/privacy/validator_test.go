package privacy

import (
	"testing"

	privacy_pb "github.com/IABTechLab/agentic-rtb-framework/pkg/pb/privacy"
)

func TestValidateConsent_Granted(t *testing.T) {
	validator := NewValidator("test-agent", false)

	ctx := &privacy_pb.PrivacyContext{
		Consent: &privacy_pb.Consent{
			Status:   privacy_pb.Consent_GRANTED,
			Purposes: []string{"measurement", "personalization"},
		},
	}

	result, err := validator.Validate(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Result.Compliant {
		t.Errorf("expected compliant=true, got false")
	}

	if result.Action.Action != privacy_pb.EnforcementAction_ALLOW {
		t.Errorf("expected ALLOW action, got %v", result.Action.Action)
	}
}

func TestValidateConsent_Denied(t *testing.T) {
	validator := NewValidator("test-agent", true)

	ctx := &privacy_pb.PrivacyContext{
		Consent: &privacy_pb.Consent{
			Status: privacy_pb.Consent_DENIED,
		},
	}

	result, err := validator.Validate(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Result.Compliant {
		t.Errorf("expected compliant=false, got true")
	}

	if result.Action.Action != privacy_pb.EnforcementAction_BLOCK {
		t.Errorf("expected BLOCK action, got %v", result.Action.Action)
	}

	if len(result.Violations) == 0 {
		t.Errorf("expected violations, got none")
	}
}

func TestValidateAgentAuthorization_Allowed(t *testing.T) {
	validator := NewValidator("test-agent", false)

	ctx := &privacy_pb.PrivacyContext{
		Consent: &privacy_pb.Consent{
			Status: privacy_pb.Consent_GRANTED,
		},
		ProcessingConstraints: &privacy_pb.ProcessingConstraints{
			AllowedAgents: []string{"test-agent", "other-agent"},
		},
	}

	result, err := validator.Validate(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Result.Compliant {
		t.Errorf("expected compliant=true, got false")
	}
}

func TestValidateAgentAuthorization_Denied(t *testing.T) {
	validator := NewValidator("test-agent", false)

	ctx := &privacy_pb.PrivacyContext{
		Consent: &privacy_pb.Consent{
			Status: privacy_pb.Consent_GRANTED,
		},
		ProcessingConstraints: &privacy_pb.ProcessingConstraints{
			AllowedAgents: []string{"other-agent"},
		},
	}

	result, err := validator.Validate(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Result.Compliant {
		t.Errorf("expected compliant=false, got true")
	}

	// Check for unauthorized agent violation
	found := false
	for _, v := range result.Violations {
		if v.Type == privacy_pb.PolicyViolation_UNAUTHORIZED_AGENT {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected UNAUTHORIZED_AGENT violation")
	}
}

func TestEnforcementAction_StripPII(t *testing.T) {
	validator := NewValidator("test-agent", false)

	personalDataAllowed := false
	ctx := &privacy_pb.PrivacyContext{
		Consent: &privacy_pb.Consent{
			Status: privacy_pb.Consent_GRANTED,
		},
		DataControls: &privacy_pb.DataControls{
			PersonalDataAllowed: &personalDataAllowed,
		},
	}

	result, err := validator.Validate(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Action.Action != privacy_pb.EnforcementAction_STRIP_PII {
		t.Errorf("expected STRIP_PII action, got %v", result.Action.Action)
	}

	if len(result.Action.AffectedFields) == 0 {
		t.Errorf("expected affected fields, got none")
	}
}

func TestEnforcementAction_Anonymize(t *testing.T) {
	validator := NewValidator("test-agent", false)

	anonymizationRequired := true
	ctx := &privacy_pb.PrivacyContext{
		Consent: &privacy_pb.Consent{
			Status: privacy_pb.Consent_GRANTED,
		},
		DataControls: &privacy_pb.DataControls{
			AnonymizationRequired: &anonymizationRequired,
		},
	}

	result, err := validator.Validate(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Action.Action != privacy_pb.EnforcementAction_ANONYMIZE {
		t.Errorf("expected ANONYMIZE action, got %v", result.Action.Action)
	}
}

func TestComplianceScore(t *testing.T) {
	validator := NewValidator("test-agent", false)

	tests := []struct {
		name       string
		violations []*privacy_pb.PolicyViolation
		wantScore  float64
	}{
		{
			name:       "no violations",
			violations: nil,
			wantScore:  1.0,
		},
		{
			name: "low severity",
			violations: []*privacy_pb.PolicyViolation{
				{Severity: 2},
			},
			wantScore: 0.8,
		},
		{
			name: "high severity",
			violations: []*privacy_pb.PolicyViolation{
				{Severity: 10},
			},
			wantScore: 0.0,
		},
		{
			name: "multiple violations",
			violations: []*privacy_pb.PolicyViolation{
				{Severity: 5},
				{Severity: 5},
			},
			wantScore: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := validator.calculateComplianceScore(tt.violations)
			if score != tt.wantScore {
				t.Errorf("expected score %f, got %f", tt.wantScore, score)
			}
		})
	}
}

func TestNilPrivacyContext(t *testing.T) {
	validator := NewValidator("test-agent", false)

	result, err := validator.Validate(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Result.Compliant {
		t.Errorf("expected compliant=true for nil context, got false")
	}

	if result.Action.Action != privacy_pb.EnforcementAction_ALLOW {
		t.Errorf("expected ALLOW action, got %v", result.Action.Action)
	}
}

func TestTelemetry(t *testing.T) {
	validator := NewValidator("test-agent", false)

	telemetryCalled := false
	var lastEvent string

	validator.SetTelemetry(func(event string, metadata map[string]interface{}) {
		telemetryCalled = true
		lastEvent = event
	})

	ctx := &privacy_pb.PrivacyContext{
		Consent: &privacy_pb.Consent{
			Status: privacy_pb.Consent_GRANTED,
		},
	}

	_, err := validator.Validate(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !telemetryCalled {
		t.Errorf("expected telemetry to be called")
	}

	if lastEvent != "privacy.consent.validated" {
		t.Errorf("expected last event 'privacy.consent.validated', got '%s'", lastEvent)
	}
}
