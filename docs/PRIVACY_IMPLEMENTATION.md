# Privacy Context Extension - Implementation Guide

This guide shows how to integrate and use the Privacy Context Extension in your Agentic RTB Framework agents.

## Quick Start

### 1. Add Privacy Context to Request

Orchestrators should include privacy context in `RTBRequest.Ext`:

```go
import (
    pb "github.com/IABTechLab/agentic-rtb-framework/pkg/pb/artf"
    privacy_pb "github.com/IABTechLab/agentic-rtb-framework/pkg/pb/privacy"
)

// Create request with privacy context
request := &pb.RTBRequest{
    Id:       "req-123",
    Tmax:     100,
    BidRequest: bidRequest,
    Ext: &pb.RTBRequest_Ext{
        PrivacyContext: &privacy_pb.PrivacyContext{
            Regulations: &privacy_pb.Regulations{
                Gdpr: true,
            },
            Consent: &privacy_pb.Consent{
                Status: privacy_pb.Consent_GRANTED,
                Purposes: []string{"measurement", "personalization"},
                ConsentString: "COtybn4Otybn4ABABBENAPCgAAAAAAAAAAAEgAAAAAAAA",
            },
            DataControls: &privacy_pb.DataControls{
                PersonalDataAllowed: &falseVal,
                RetentionTtlMs:      &retention,
            },
        },
    },
}
```

### 2. Validate Privacy Context in Agent

```go
import (
    "github.com/IABTechLab/agentic-rtb-framework/internal/privacy"
)

func (s *AgentServer) GetMutations(ctx context.Context, req *pb.RTBRequest) (*pb.RTBResponse, error) {
    // Create validator
    validator := privacy.NewValidator("my-agent-id", true)
    
    // Extract privacy context
    privacyCtx := req.Ext.GetPrivacyContext()
    
    // Validate
    result, err := validator.Validate(privacyCtx)
    if err != nil {
        return nil, err
    }
    
    // Check enforcement action
    switch result.Action.Action {
    case privacy_pb.EnforcementAction_BLOCK:
        // Return early with validation result
        return &pb.RTBResponse{
            Id: req.Id,
            Mutations: []*pb.Mutation{
                {
                    Intent: pb.Intent_VALIDATE_PRIVACY,
                    Op:     pb.Operation_OPERATION_ADD,
                    Path:   "/privacy/validation",
                    Value: &pb.Mutation_PrivacyValidation{
                        PrivacyValidation: result,
                    },
                },
            },
        }, nil
        
    case privacy_pb.EnforcementAction_STRIP_PII:
        // Strip PII before processing
        privacy.StripPII(req.BidRequest)
    }
    
    // Continue with normal mutation processing...
    return processNormalMutations(ctx, req)
}
```

### 3. Use Privacy-Aware Handler Wrapper

For cleaner code, use the provided handler wrapper:

```go
import (
    "github.com/IABTechLab/agentic-rtb-framework/internal/handlers"
)

// Create base handler
baseHandler := handlers.NewSegmentActivationHandler("audience-agent")

// Wrap with privacy enforcement
privacyHandler := handlers.NewPrivacyAwareHandler("audience-agent", true, baseHandler)

// Configure telemetry (optional)
privacyHandler.SetTelemetry(func(event string, metadata map[string]interface{}) {
    log.Printf("Privacy event: %s - %v", event, metadata)
})

// Use in gRPC server
func (s *AgentServer) GetMutations(ctx context.Context, req *pb.RTBRequest) (*pb.RTBResponse, error) {
    return privacyHandler.GetMutations(ctx, req)
}
```

## Configuration Options

### Strict Mode

When `strictMode = true`:
- `consent.status = "unknown"` is treated as violation
- All privacy constraints are strictly enforced

When `strictMode = false`:
- `consent.status = "unknown"` is allowed to proceed
- More permissive validation

### Telemetry Events

The privacy validator emits the following events:

| Event | When Fired | Metadata |
|-------|-----------|----------|
| `privacy.context.received` | Privacy context received | `agent_id` |
| `privacy.consent.validated` | Consent validation passed | `agent_id`, `score` |
| `privacy.policy.violation` | Privacy violation detected | `agent_id`, `violation_count`, `action` |
| `privacy.pii.stripped` | PII removed from request | `agent_id` |
| `privacy.execution.restricted` | Request blocked | `agent_id`, `reason` |

## Example Scenarios

### GDPR-Compliant Request

```json
{
  "privacy_context": {
    "regulations": {
      "gdpr": true
    },
    "consent": {
      "status": "granted",
      "purposes": ["measurement", "frequency_capping"],
      "consent_string": "COtybn4Otybn4ABABBENAPCgAAAAAAAAAAAEgAAAAAAAA"
    },
    "data_controls": {
      "personal_data_allowed": false,
      "retention_ttl_ms": 7776000000
    },
    "processing_constraints": {
      "geo_scope": "EEA",
      "prohibited_enrichments": ["precise_location"]
    }
  }
}
```

**Agent Behavior:**
1. Validates consent is granted
2. Strips PII fields (`user.id`, `device.ifa`)
3. Ensures data deleted after 90 days
4. Blocks location-based enrichment

### CCPA Opt-Out

```json
{
  "privacy_context": {
    "regulations": {
      "ccpa": true
    },
    "consent": {
      "status": "denied"
    }
  }
}
```

**Agent Behavior:**
1. Detects `consent.status = "denied"`
2. Returns `BLOCK` enforcement action
3. Returns validation mutation with violations
4. No user-level processing occurs

### Contextual-Only Auction

```json
{
  "privacy_context": {
    "consent": {
      "status": "not_required"
    },
    "processing_constraints": {
      "allowed_agents": ["contextual-agent"],
      "prohibited_enrichments": ["user_data", "cross_site_tracking"]
    }
  }
}
```

**Agent Behavior:**
1. Only contextual signals allowed
2. User-level data enrichment blocked
3. Returns `CONTEXTUAL_ONLY` enforcement action

## Testing

Run the privacy validator tests:

```bash
cd internal/privacy
go test -v
```

Example test output:

```
=== RUN   TestValidateConsent_Granted
--- PASS: TestValidateConsent_Granted (0.00s)
=== RUN   TestValidateConsent_Denied
--- PASS: TestValidateConsent_Denied (0.00s)
=== RUN   TestValidateAgentAuthorization_Allowed
--- PASS: TestValidateAgentAuthorization_Allowed (0.00s)
=== RUN   TestEnforcementAction_StripPII
--- PASS: TestEnforcementAction_StripPII (0.00s)
PASS
```

## Integration Checklist

- [ ] Add `privacy_context.proto` to your protobuf compilation
- [ ] Import privacy package in your agent code
- [ ] Create privacy validator with your agent ID
- [ ] Extract privacy context from `RTBRequest.Ext`
- [ ] Validate privacy context before processing
- [ ] Handle enforcement actions appropriately
- [ ] Add privacy validation mutation to response
- [ ] Configure telemetry (optional)
- [ ] Test with sample privacy contexts
- [ ] Document privacy compliance in agent manifest

## Protobuf Compilation

Generate Go code from proto files:

```bash
protoc \
  --go_out=pkg/pb \
  --go_opt=paths=source_relative \
  --go-grpc_out=pkg/pb \
  --go-grpc_opt=paths=source_relative \
  proto/privacy_context.proto \
  proto/agenticrtbframework.proto \
  proto/agenticrtbframeworkservices.proto
```

Or use the Makefile:

```bash
make generate
```

## Further Reading

- [Privacy Context Extension Specification](./privacy_context_extension.md)
- [IAB TCF Documentation](https://iabeurope.eu/tcf-2-0/)
- [GDPR Guidance](https://gdpr.eu/)
- [CCPA/CPRA Regulations](https://oag.ca.gov/privacy/ccpa)

## Support

For questions or issues:
- **GitHub Issues**: [Open an issue](https://github.com/IABTechLab/agentic-rtb-framework/issues)
- **Email**: support@iabtechlab.com

## License

This implementation is licensed under the same license as the Agentic RTB Framework.

See [LICENSE](../LICENSE) for details.
