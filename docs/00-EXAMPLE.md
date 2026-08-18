# Agentic RTB Framework - Implementation Specification

This document provides a comprehensive specification for the Go reference implementation of the IAB Tech Lab's Agentic RTB Framework (ARTF) v1.0.

---

## Table of Contents

1. [Overview](#overview)
2. [Protocol Definition](#protocol-definition)
3. [Project Structure](#project-structure)
4. [API Specification](#api-specification)
5. [Message Types](#message-types)
6. [Intents and Operations](#intents-and-operations)
7. [Server Endpoints](#server-endpoints)
8. [Container Requirements](#container-requirements)
9. [Build and Deployment](#build-and-deployment)
10. [Configuration](#configuration)

---

## Overview

The Agentic RTB Framework (ARTF) defines a standard for implementing agent services that operate within a host platform's infrastructure to process bidstream data in real-time. This implementation provides:

- **Segment Activation** - Activate user segments based on bid request data
- **Deal Management** - Activate, suppress, and adjust deals dynamically
- **Bid Shading** - Optimize bid prices using intelligent pricing strategies
- **Metrics Addition** - Add viewability and other metrics to impressions

### Key Principles

1. **Agents participate in the core bidstream** - Real-time transaction processing
2. **Agents accomplish specific goals** - Each agent declares specific intents
3. **Agents are composable and deployable** - OCI containers, Kubernetes-ready
4. **Agents are performant** - gRPC/protobuf, sub-millisecond latency
5. **Agents follow least-privilege** - No external network access, minimal data exposure

---

## Protocol Definition

### Service Definition

```protobuf
syntax = "proto2";
package com.iabtechlab.bidstream.mutation.v1;

service RTBExtensionPoint {
  // GetMutations returns RTBResponse containing mutations to be applied
  // at the predetermined auction lifecycle event
  rpc GetMutations (RTBRequest) returns (RTBResponse);
}
```

### Wire Protocol

| Attribute | Value |
|-----------|-------|
| Protocol | gRPC over HTTP/2 |
| Serialization | Protocol Buffers (proto2) |
| Default Port | 50051 |
| TLS | Recommended for production |

---

## Project Structure

```
agentic-rtb-framework/
├── cmd/
│   └── server/
│       └── main.go              # Server entry point
├── internal/
│   ├── handlers/
│   │   └── handlers.go          # Mutation handlers
│   ├── health/
│   │   └── health.go            # Health check endpoints
│   └── server/
│       └── server.go            # gRPC service implementation
├── pkg/
│   └── pb/                      # Generated protobuf code
│       ├── artf/                # ARTF messages and service
│       └── openrtb/             # OpenRTB v2.6 messages
├── proto/
│   ├── agenticrtbframework.proto
│   └── com/iabtechlab/openrtb/v2.6/
│       └── openrtb.proto
├── scripts/
│   └── generate.sh              # Protobuf generation script
├── docs/
│   └── SPECIFICATION.md         # This file
├── CLAUDE.md                    # AI assistant instructions
├── README.md                    # Project readme
├── Dockerfile                   # Container build
├── docker-compose.yml           # Local development
├── Makefile                     # Build automation
├── go.mod                       # Go module definition
└── go.sum                       # Dependency checksums
```

---

## API Specification

### RTBRequest

The request message sent from the orchestrator to the agent.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `lifecycle` | Lifecycle | Yes | Auction lifecycle stage |
| `id` | string | Yes | Unique request ID assigned by exchange |
| `tmax` | int32 | Yes | Maximum response time in milliseconds |
| `bid_request` | BidRequest | Yes | OpenRTB v2.6 bid request |
| `bid_response` | BidResponse | No | OpenRTB v2.6 bid response (if available) |
| `ext` | Extensions | No | Extension fields |

### RTBResponse

The response message returned by the agent.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Request ID (must match request) |
| `mutations` | Mutation[] | No | List of proposed mutations |
| `metadata` | Metadata | No | Response metadata |

### Mutation

A single atomic change proposed to the bid request or response.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `intent` | Intent | Yes | Purpose of the mutation |
| `op` | Operation | Yes | Operation type (add/remove/replace) |
| `path` | string | Yes | Semantic path to target data |
| `value` | oneof | Yes | Payload (type depends on intent) |

### Metadata

Optional metadata about the response.

| Field | Type | Description |
|-------|------|-------------|
| `api_version` | string | Version of the agent API |
| `model_version` | string | Version of the ML model (if applicable) |

---

## Message Types

### Payload Types

#### IDsPayload

Used for segment and deal ID lists.

```protobuf
message IDsPayload {
  repeated string id = 1;
}
```

#### AdjustDealPayload

Used for deal floor and margin adjustments.

```protobuf
message AdjustDealPayload {
  optional double bidfloor = 1;
  optional Margin margin = 2;
}

message Margin {
  optional double value = 1;
  optional CalculationType calculation_type = 2;

  enum CalculationType {
    CPM = 0;      // Absolute margin
    PERCENT = 1;  // Relative margin (percentage)
  }
}
```

#### AdjustBidPayload

Used for bid price adjustments.

```protobuf
message AdjustBidPayload {
  optional double price = 1;
}
```

#### AddMetricsPayload

Used for adding impression metrics.

```protobuf
message AddMetricsPayload {
  repeated Metric metric = 1;  // OpenRTB Metric objects
}
```

---

## Intents and Operations

### Intent Enum

| Value | Name | Description |
|-------|------|-------------|
| 0 | `INTENT_UNSPECIFIED` | Unspecified (invalid) |
| 1 | `ACTIVATE_SEGMENTS` | Activate user segments by external segment IDs |
| 2 | `ACTIVATE_DEALS` | Activate deals by external deal IDs |
| 3 | `SUPPRESS_DEALS` | Suppress deals by external deal IDs |
| 4 | `ADJUST_DEAL_FLOOR` | Adjust the bid floor of a specific deal |
| 5 | `ADJUST_DEAL_MARGIN` | Adjust the deal margin of a specific deal |
| 6 | `BID_SHADE` | Adjust the bid price of a specific bid |
| 7 | `ADD_METRICS` | Add metrics to an impression |

### Operation Enum

| Value | Name | Description |
|-------|------|-------------|
| 0 | `OPERATION_UNSPECIFIED` | Unspecified (invalid) |
| 1 | `OPERATION_ADD` | Add new data to the target |
| 2 | `OPERATION_REMOVE` | Remove data from the target |
| 3 | `OPERATION_REPLACE` | Replace existing data at the target |

### Intent-Payload Mapping

| Intent | Expected Payload | Path Example |
|--------|-----------------|--------------|
| `ACTIVATE_SEGMENTS` | IDsPayload | `/user/data/segment` |
| `ACTIVATE_DEALS` | IDsPayload | `/imp/{id}` |
| `SUPPRESS_DEALS` | IDsPayload | `/imp/{id}` |
| `ADJUST_DEAL_FLOOR` | AdjustDealPayload | `/imp/{id}/pmp/deals/{dealId}` |
| `ADJUST_DEAL_MARGIN` | AdjustDealPayload | `/imp/{id}/pmp/deals/{dealId}` |
| `BID_SHADE` | AdjustBidPayload | `/seatbid/{seat}/bid/{bidId}` |
| `ADD_METRICS` | AddMetricsPayload | `/imp/{id}/metric` |

---

## Server Endpoints

### gRPC Service

| Port | Service | Method | Description |
|------|---------|--------|-------------|
| 50051 | RTBExtensionPoint | GetMutations | Process bid request and return mutations |

### Health Check HTTP Endpoints

| Port | Path | Method | Description |
|------|------|--------|-------------|
| 8080 | `/health/live` | GET | Liveness probe - returns 200 if process is alive |
| 8080 | `/health/ready` | GET | Readiness probe - returns 200 if ready for traffic |

### Health Response Format

```json
{
  "status": "ready",
  "ready": true,
  "version": "1.0.0"
}
```

---

## Container Requirements

### Security Requirements (per ARTF spec)

| Requirement | Implementation |
|-------------|----------------|
| Non-root user | `USER 65534:65534` in Dockerfile |
| Drop capabilities | `cap_drop: ALL` in docker-compose |
| Read-only filesystem | `read_only: true` in docker-compose |
| No privilege escalation | `no-new-privileges: true` |
| Network isolation | Configurable via network policies |

### Resource Defaults

| Resource | Default | Description |
|----------|---------|-------------|
| CPU Limit | 500m | 0.5 CPU cores |
| Memory Limit | 256Mi | 256 MB RAM |
| CPU Request | 250m | 0.25 CPU cores |
| Memory Request | 128Mi | 128 MB RAM |

### Health Probes

```yaml
livenessProbe:
  httpGet:
    path: /health/live
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 5

readinessProbe:
  httpGet:
    path: /health/ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
```

---

## Build and Deployment

### Prerequisites

- Go 1.22+
- Protocol Buffers compiler (`protoc`)
- protoc-gen-go and protoc-gen-go-grpc plugins
- Docker (for containerized deployment)

### Build Commands

| Command | Description |
|---------|-------------|
| `make deps` | Download Go dependencies |
| `make generate` | Confirm vendored protos / checked-in `pkg/pb/` (does not require protoc) |
| `make build` | Build the Go agent binary |
| `make build-rust` | Build the Rust reference service (`rust/Cargo.toml`) |
| `make test` | Run unit tests |
| `make test-coverage` | Run tests with coverage report |
| `make lint` | `go vet ./...` |
| `make clean` | Remove build artifacts (`artf-agent`, `rust/target`, coverage files) |

### Run Commands

| Command | Description |
|---------|-------------|
| `make run` | Run server locally (gRPC + MCP + web) |
| `make run-all` | Same as `make run` |
| `make docker-build` | Build Docker image |
| `make docker-run` | Run Docker container |
| `make docker-compose-up` | Start with docker-compose |
| `make docker-compose-down` | Stop docker-compose services |

### Testing Commands

| Command | Description |
|---------|-------------|
| `make grpc-test` | Test gRPC endpoint with grpcurl |
| `make health-check` | Check health endpoints with curl |

---

## Configuration

### Command-Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-grpc-port` | 50051 | gRPC server listening port |
| `-health-port` | 8080 | Health check HTTP server port |
| `-health-check` | false | Probe local `/health/ready` and exit 0/1 (container HEALTHCHECK) |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `GRPC_PORT` | Override gRPC port |
| `HEALTH_PORT` | Override health check port |

### Agent Manifest (Container Label)

The container image must include an `agent-manifest` label with JSON metadata:

```json
{
  "name": "artf-agent",
  "version": "1.0.0",
  "vendor": "your-organization",
  "owner": "team@example.com",
  "resources": {
    "cpu": "500m",
    "memory": "256Mi"
  },
  "intents": [
    "ACTIVATE_SEGMENTS",
    "ACTIVATE_DEALS",
    "SUPPRESS_DEALS",
    "ADJUST_DEAL_FLOOR",
    "BID_SHADE"
  ],
  "dependencies": {},
  "health": {
    "livenessProbe": {
      "httpGet": { "path": "/health/live", "port": 8080 }
    },
    "readinessProbe": {
      "httpGet": { "path": "/health/ready", "port": 8080 }
    }
  },
  "security": {
    "runAsNonRoot": true,
    "dropCapabilities": ["NET_ADMIN", "SYS_PTRACE"]
  }
}
```

---

## Example Mutations

### Activate Segments

```json
{
  "intent": "ACTIVATE_SEGMENTS",
  "op": "OPERATION_ADD",
  "path": "/user/data/segment",
  "ids": {
    "id": ["demo-18-24", "sports-enthusiast", "premium-user"]
  }
}
```

### Activate Deals

```json
{
  "intent": "ACTIVATE_DEALS",
  "op": "OPERATION_ADD",
  "path": "/imp/1",
  "ids": {
    "id": ["deal-001", "deal-002"]
  }
}
```

### Adjust Deal Floor

```json
{
  "intent": "ADJUST_DEAL_FLOOR",
  "op": "OPERATION_REPLACE",
  "path": "/imp/1/pmp/deals/deal-001",
  "adjust_deal": {
    "bidfloor": 5.50
  }
}
```

### Bid Shading

```json
{
  "intent": "BID_SHADE",
  "op": "OPERATION_REPLACE",
  "path": "/seatbid/seat-1/bid/bid-123",
  "adjust_bid": {
    "price": 4.25
  }
}
```

---

## References

- [IAB Tech Lab Agentic RTB Framework v1.0](https://iabtechlab.com/standards/artf/)
- [OpenRTB v2.6 Specification](https://iabtechlab.com/standards/openrtb/)
- [gRPC Documentation](https://grpc.io/docs/)
- [Protocol Buffers](https://protobuf.dev/)
- [OCI Container Specification](https://opencontainers.org/)

---

*Document Version: 1.0.0*
*Last Updated: November 2025*
