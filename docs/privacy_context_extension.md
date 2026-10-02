# Privacy Context Extension Specification

**Version:** 1.0  
**Status:** Proposed Extension — Community Review Requested  
**Last Updated:** February 2026  

---

## Overview

The **Privacy Context Extension** defines a standardized mechanism for propagating privacy, consent, and data usage constraints between agents participating in Agentic Audience and Agentic RTB workflows.

As agent-based advertising systems exchange identity signals, embeddings, and audience metadata, privacy intent **MUST** travel alongside data to ensure compliant processing across distributed agent execution environments.

This extension introduces a portable `privacy_context` object that enables:

- **Consent-aware agent execution**
- **Runtime data minimization**
- **Regional regulatory enforcement**
- **Auditable decision pipelines**
- **Inter-agent privacy interoperability**

---

## Design Goals

- **Preserve user privacy intent across agent boundaries**
- **Enable Privacy-by-Design execution models**
- **Maintain backward compatibility** with existing schemas
- **Support low-latency real-time bidding** environments
- **Allow extensibility** for regional regulations

---

## Integration with Agentic RTB Framework

This extension integrates with the existing ARTF specification by:

1. **Extending `RTBRequest.Ext`** to include privacy context metadata
2. **Introducing a new Intent** (`VALIDATE_PRIVACY`) for privacy enforcement
3. **Adding validation handlers** in the agent processing pipeline
4. **Ensuring mutation compliance** with privacy constraints

---

## Privacy Context Object

### Schema Definition

The `privacy_context` object MAY be included within the `RTBRequest.Ext` extension field.

```json
{
  "privacy_context": {
    "regulations": {
      "gdpr": true,
      "ccpa": true,
      "cpra": false,
      "lgpd": false
    },
    "consent": {
      "status": "granted",
      "purposes": [
        "measurement",
        "frequency_capping",
        "personalization"
      ],
      "consent_string": "COtybn4Otybn4ABABBENAPCgAAAAAAAAAAAEgAAAAAAAA",
      "timestamp_ms": 1710000000000
    },
    "data_controls": {
      "data_minimization": true,
      "personal_data_allowed": false,
      "retention_ttl_ms": 300000,
      "anonymization_required": true
    },
    "processing_constraints": {
      "geo_scope": "EEA",
      "allowed_agents": [
        "audience-agent",
        "measurement-agent"
      ],
      "prohibited_enrichments": [
        "precise_location",
        "biometric_data"
      ]
    },
    "audit": {
      "consent_verified": true,
      "verification_timestamp_ms": 1710000000000,
      "verifier_id": "consent-service-v2"
    }
  }
}
```

---

## Field Definitions

### Regulations Object

Indicates which privacy regulations apply to this request.

| Field    | Type    | Required | Description                                      |
|----------|---------|----------|--------------------------------------------------|
| `gdpr`   | boolean | No       | European Union General Data Protection Regulation |
| `ccpa`   | boolean | No       | California Consumer Privacy Act                   |
| `cpra`   | boolean | No       | California Privacy Rights Act                     |
| `lgpd`   | boolean | No       | Brazilian General Data Protection Law             |

### Consent Object

Represents user consent status and scope.

| Field             | Type     | Required | Description                                                    |
|-------------------|----------|----------|----------------------------------------------------------------|
| `status`          | enum     | Yes      | Consent status: `granted`, `denied`, `unknown`, `not_required` |
| `purposes`        | string[] | No       | Approved processing purposes (e.g., `personalization`)          |
| `consent_string`  | string   | No       | Encoded consent string (e.g., IAB TCF format)                  |
| `timestamp_ms`    | int64    | No       | Unix timestamp (milliseconds) when consent was captured        |

#### Consent Status Values

| Value          | Description                                          |
|----------------|------------------------------------------------------|
| `granted`      | User has explicitly granted consent                  |
| `denied`       | User has explicitly denied consent                   |
| `unknown`      | Consent status could not be determined               |
| `not_required` | Consent not required under applicable law            |

### Data Controls Object

Specifies data handling and minimization requirements.

| Field                     | Type    | Required | Description                                                |
|---------------------------|---------|----------|------------------------------------------------------------|
| `data_minimization`       | boolean | No       | Indicates reduced data exposure is required                |
| `personal_data_allowed`   | boolean | No       | Whether personally identifiable information (PII) is allowed |
| `retention_ttl_ms`        | int64   | No       | Maximum data retention period (milliseconds)               |
| `anonymization_required`  | boolean | No       | Whether data must be anonymized before processing          |

### Processing Constraints Object

Defines execution boundaries and restrictions for agents.

| Field                      | Type     | Required | Description                                                  |
|----------------------------|----------|----------|--------------------------------------------------------------|
| `geo_scope`                | string   | No       | Geographic processing boundary (e.g., `EEA`, `US`, `BR`)     |
| `allowed_agents`           | string[] | No       | Whitelist of agents permitted to access this data            |
| `prohibited_enrichments`   | string[] | No       | Types of data enrichment explicitly forbidden                |

### Audit Object

Provides traceability for consent verification.

| Field                        | Type    | Required | Description                                              |
|------------------------------|---------|----------|----------------------------------------------------------|
| `consent_verified`           | boolean | No       | Whether upstream consent verification occurred           |
| `verification_timestamp_ms`  | int64   | No       | Unix timestamp (milliseconds) of verification            |
| `verifier_id`                | string  | No       | Identifier of the service that verified consent          |

---

## Protobuf Schema Extension

### Extended `RTBRequest.Ext`

To integrate privacy context into the existing ARTF protobuf schema, extend the `RTBRequest.Ext` message:

```protobuf
message Ext {
  // Existing extension fields
  extensions 500 to max;
  
  // Privacy Context Extension (field number in reserved extension range)
  optional PrivacyContext privacy_context = 600;
}

message PrivacyContext {
  optional Regulations regulations = 1;
  optional Consent consent = 2;
  optional DataControls data_controls = 3;
  optional ProcessingConstraints processing_constraints = 4;
  optional Audit audit = 5;
}

message Regulations {
  optional bool gdpr = 1;
  optional bool ccpa = 2;
  optional bool cpra = 3;
  optional bool lgpd = 4;
}

message Consent {
  enum Status {
    STATUS_UNSPECIFIED = 0;
    GRANTED = 1;
    DENIED = 2;
    UNKNOWN = 3;
    NOT_REQUIRED = 4;
  }
  
  optional Status status = 1;
  repeated string purposes = 2;
  optional string consent_string = 3;
  optional int64 timestamp_ms = 4;
}

message DataControls {
  optional bool data_minimization = 1;
  optional bool personal_data_allowed = 2;
  optional int64 retention_ttl_ms = 3;
  optional bool anonymization_required = 4;
}

message ProcessingConstraints {
  optional string geo_scope = 1;
  repeated string allowed_agents = 2;
  repeated string prohibited_enrichments = 3;
}

message Audit {
  optional bool consent_verified = 1;
  optional int64 verification_timestamp_ms = 2;
  optional string verifier_id = 3;
}
```

---

## Agent Responsibilities

### MUST Requirements

Agents receiving a `privacy_context` **MUST**:

1. **Validate consent** before processing user-linked signals
2. **Restrict enrichment operations** outside approved purposes
3. **Respect retention** and geographic constraints
4. **Propagate unchanged** privacy metadata downstream
5. **Log enforcement outcomes** when audit logging is enabled

### MUST NOT Requirements

Agents **MUST NOT**:

- Expand permissions beyond received constraints
- Process data when `consent.status = "denied"`
- Access PII when `data_controls.personal_data_allowed = false`
- Execute outside specified `geo_scope`
- Perform enrichments listed in `prohibited_enrichments`

---

## Validation Reference Implementation

### Go Example

```go
package privacy

import (
    "errors"
    pb "github.com/IABTechLab/agentic-rtb-framework/pkg/pb"
)

// ValidatePrivacyContext enforces privacy constraints before agent execution
func ValidatePrivacyContext(ctx *pb.PrivacyContext) error {
    if ctx == nil {
        return nil // Privacy context is optional
    }
    
    // Enforce consent requirements
    if ctx.Consent != nil {
        switch ctx.Consent.Status {
        case pb.Consent_DENIED:
            return errors.New("privacy: consent denied")
        case pb.Consent_UNKNOWN:
            // Policy decision: strict mode requires explicit consent
            return errors.New("privacy: consent status unknown")
        }
    }
    
    // Enforce data controls
    if ctx.DataControls != nil {
        if ctx.DataControls.PersonalDataAllowed != nil && !*ctx.DataControls.PersonalDataAllowed {
            // Strip PII fields from bid request before processing
            return nil // Indicate PII stripping required
        }
    }
    
    return nil
}

// EnforceRetention checks data retention limits
func EnforceRetention(ctx *pb.PrivacyContext, dataTimestampMs int64, currentTimeMs int64) error {
    if ctx == nil || ctx.DataControls == nil || ctx.DataControls.RetentionTtlMs == nil {
        return nil
    }
    
    age := currentTimeMs - dataTimestampMs
    if age > *ctx.DataControls.RetentionTtlMs {
        return errors.New("privacy: data retention limit exceeded")
    }
    
    return nil
}
```

---

## Integration Points

### Agentic Audiences

- **Attach** `privacy_context` to audience embeddings
- **Enforce** purpose-scoped activation
- **Validate** consent before lookalike modeling

### Agentic RTB Framework

- **Validate** privacy context in `GetMutations` RPC handler
- **Apply** execution gating during auctions
- **Reject** non-compliant mutations

### Seller Agent

- **Normalize** upstream consent signals
- **Reject** non-compliant enrichment requests
- **Audit** privacy enforcement decisions

---

## New Intent: VALIDATE_PRIVACY

To support privacy validation as a first-class operation, add a new intent to the ARTF specification:

```protobuf
enum Intent {
  // ... existing intents ...
  
  // Validate privacy context and enforce constraints
  VALIDATE_PRIVACY = 9;
}
```

### Usage

Agents can propose privacy validation mutations to signal enforcement:

```json
{
  "intent": "VALIDATE_PRIVACY",
  "op": "OPERATION_ADD",
  "path": "/privacy/validation",
  "value": {
    "validation_result": {
      "compliant": true,
      "violations": [],
      "enforcement_action": "allow"
    }
  }
}
```

---

## Observability (Optional)

Recommended telemetry events for privacy enforcement:

| Event Name                    | Description                                          |
|-------------------------------|------------------------------------------------------|
| `privacy.context.received`    | Privacy context received in request                  |
| `privacy.consent.validated`   | Consent validation completed                         |
| `privacy.execution.restricted`| Request blocked due to privacy constraints           |
| `privacy.policy.violation`    | Privacy policy violation detected                    |
| `privacy.pii.stripped`        | PII removed from request before agent processing     |

---

## Backward Compatibility

Agents that do not recognize this extension **MUST** safely ignore the `privacy_context` object without altering existing execution behavior.

The extension uses the reserved extension field range (`500 to max`) to ensure forward compatibility with future ARTF versions.

---

## Security Considerations

- **Cryptographic Signing**: Privacy context **SHOULD** be cryptographically signed when transmitted across trust boundaries
- **Immutability**: Consent strings **SHOULD NOT** be modified by downstream agents
- **Audit Integrity**: Audit metadata **SHOULD** be immutable once verified
- **Transport Security**: Privacy context **MUST** be transmitted over TLS/gRPC secure channels

---

## Future Extensions

Potential additions include:

- **Differential privacy budgets** for aggregated queries
- **PET execution flags** (e.g., secure multi-party computation)
- **Federated identity constraints** for cross-context learning
- **Zero-party data indicators** for first-party consent
- **Confidential compute attestation** for TEE environments

---

## Example Scenarios

### Scenario 1: GDPR Compliance in EEA

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
- Strip PII fields (email, IDFA) before processing
- Reject location-based enrichment
- Ensure data deleted after 90 days
- Only activate approved purposes (measurement, frequency capping)

### Scenario 2: CCPA Opt-Out

```json
{
  "privacy_context": {
    "regulations": {
      "ccpa": true
    },
    "consent": {
      "status": "denied"
    },
    "data_controls": {
      "data_minimization": true,
      "personal_data_allowed": false
    }
  }
}
```

**Agent Behavior:**
- Block all user-linked processing
- Return early without mutations
- Log enforcement action for audit trail

### Scenario 3: Contextual-Only Auction

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
- Only contextual signals (page content, keywords) allowed
- User-level data enrichment rejected
- No cross-site tracking permitted

---

## References

- [IAB Tech Lab Agentic RTB Framework v1.0](https://iabtechlab.com/standards/artf/)
- [IAB Transparency & Consent Framework (TCF)](https://iabeurope.eu/tcf-2-0/)
- [GDPR Recital 32: Consent](https://gdpr.eu/recital-32-conditions-for-consent/)
- [CCPA / CPRA Regulations](https://oag.ca.gov/privacy/ccpa)
- [OpenRTB v2.6 Specification](https://iabtechlab.com/standards/openrtb/)

---

## Contributing

This specification is open for community feedback. To propose changes:

1. **Fork** the [agentic-rtb-framework](https://github.com/IABTechLab/agentic-rtb-framework) repository
2. **Create** a feature branch: `git checkout -b privacy-context-feedback`
3. **Submit** a Pull Request with your proposed changes
4. **Discuss** in the PR comments

For questions or discussion, contact:
- **Email**: support@iabtechlab.com
- **GitHub Issues**: [Open an issue](https://github.com/IABTechLab/agentic-rtb-framework/issues)

---

## License

This specification is licensed under a [Creative Commons Attribution 3.0 License](http://creativecommons.org/licenses/by/3.0/).

By submitting contributions to this specification, you agree to license your contributions under the same Creative Commons Attribution 3.0 License.

---

**Document Version:** 1.0.0  
**Status:** Proposed Extension  
**Authors:** Contributors to IAB Tech Lab Agentic RTB Framework  
**Last Updated:** February 2026
