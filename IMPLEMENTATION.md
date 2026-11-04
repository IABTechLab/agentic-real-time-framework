# Implementation Guidance

Non-normative guidance for implementers of the [Agentic Real Time Framework specification](SPECIFICATION.md).

---

## Table of Contents

1. [Go Implementation](#go-implementation)
2. [MCP Integration](#mcp-integration)
3. [Intents](#intents)
   - [ACTIVATE_SEGMENTS](#activate_segments)
   - [ACTIVATE_DEALS](#activate_deals)
   - [SUPPRESS_DEALS](#suppress_deals)
   - [ADJUST_DEAL_FLOOR](#adjust_deal_floor)
   - [ADJUST_DEAL_MARGIN](#adjust_deal_margin)
   - [BID_SHADE](#bid_shade)
   - [ADD_METRICS](#add_metrics)
   - [ADD_CIDS](#add_cids)
   - [ADD_EMBEDDINGS](#add_embeddings)
4. [JSON Examples](#json-examples)
   - [banner-basic.json](samples/banner-basic.json)
   - [bid-shading.json](samples/bid-shading.json)
   - [multi-impression.json](samples/multi-impression.json)
   - [native-ad.json](samples/native-ad.json)
   - [video-deals.json](samples/video-deals.json)
5. [Scripts](#scripts)
   - [generate.sh](scripts/generate.sh)


---

## Go Implementation

This document provides a comprehensive specification for the Go reference implementation of the IAB Tech Lab's Agentic RTB Framework (ARTF) v1.0.

---

### Table of Contents

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

### Overview

The Agentic RTB Framework (ARTF) defines a standard for implementing agent services that operate within a host platform's infrastructure to process bidstream data in real-time. This implementation provides:

- **Segment Activation** - Activate user segments based on bid request data
- **Deal Management** - Activate, suppress, and adjust deals dynamically
- **Bid Shading** - Optimize bid prices using intelligent pricing strategies
- **Metrics Addition** - Add viewability and other metrics to impressions
- **Embeddings** - Attach Agentic Audiences data segments with required embeddings to the bid request

#### Key Principles

1. **Agents participate in the core bidstream** - Real-time transaction processing
2. **Agents accomplish specific goals** - Each agent declares specific intents
3. **Agents are composable and deployable** - OCI containers, Kubernetes-ready
4. **Agents are performant** - gRPC/protobuf, sub-millisecond latency
5. **Agents follow least-privilege** - No external network access, minimal data exposure

---

### Protocol Definition

#### Service Definition

```protobuf
edition = "2023";
package com.iabtechlab.bidstream.mutation.services.v1;

import "agenticrtbframework.proto";

service RTBExtensionPoint {
  // GetMutations returns RTBResponse containing mutations to be applied at the predetermined auction lifecycle event
  rpc GetMutations (com.iabtechlab.bidstream.mutation.v1.RTBRequest) returns (com.iabtechlab.bidstream.mutation.v1.RTBResponse);
}
```

#### Wire Protocol

| Attribute | Value |
|-----------|-------|
| Protocol | gRPC over HTTP/2 |
| Serialization | Protocol Buffers (proto3) |
| Default Port | 50051 |
| TLS | Recommended for production |

---

### Project Structure

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

### API Specification

#### RTBRequest

The request message sent from the orchestrator to the agent.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `lifecycle` | Lifecycle | Yes | Auction lifecycle stage |
| `id` | string | Yes | Unique request ID assigned by exchange |
| `tmax` | int32 | Yes | Maximum response time in milliseconds |
| `bid_request` | BidRequest | Yes | OpenRTB v2.6 bid request |
| `bid_response` | BidResponse | No | OpenRTB v2.6 bid response (if available) |
| `originator` | Originator | No | Business entity that created the request or response |
| `applicable_intents` | Intent[] | No | Intents the server is eligible to return |
| `ext` | Extensions | No | Extension fields |

#### Lifecycle

| Value | Name | Description |
|-------|------|-------------|
| 0 | `LIFECYCLE_UNSPECIFIED` | Unspecified |
| 1 | `LIFECYCLE_PUBLISHER_BID_REQUEST` | Publisher bid request stage |
| 2 | `LIFECYCLE_DSP_BID_RESPONSE` | DSP bid response stage |

#### Originator

| Value | Name | Description |
|-------|------|-------------|
| 0 | `TYPE_UNSPECIFIED` | Unspecified |
| 1 | `TYPE_PUBLISHER` | Publisher |
| 2 | `TYPE_SSP` | SSP |
| 3 | `TYPE_EXCHANGE` | Exchange |
| 4 | `TYPE_DSP` | DSP |

#### RTBResponse

The response message returned by the agent.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Request ID (must match request) |
| `mutations` | Mutation[] | No | List of proposed mutations |
| `metadata` | Metadata | No | Response metadata |

#### Mutation

A single atomic change proposed to the bid request or response.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `intent` | Intent | Yes | Purpose of the mutation |
| `op` | Operation | Yes | Operation type (add/remove/replace) |
| `path` | string | Yes | Semantic path to target data |
| `value` | oneof | Yes | Payload (type depends on intent) |

#### Metadata

Optional metadata about the response.

| Field | Type | Description |
|-------|------|-------------|
| `api_version` | string | Version of the agent API |
| `model_version` | string | Version of the ML model (if applicable) |

---

### Message Types

#### Payload Types

##### IDsPayload

Used for segment and deal ID lists.

```protobuf
message IDsPayload {
  repeated string id = 1;
}
```

##### AdjustDealPayload

Used for deal floor and margin adjustments.

```protobuf
message AdjustDealPayload {
  double bidfloor = 1;
  Margin margin = 2;
}

message Margin {
  double value = 1;
  CalculationType calculation_type = 2;

  enum CalculationType {
    CPM = 0;      // Absolute margin
    PERCENT = 1;  // Relative margin (percentage)
  }
}
```

##### AdjustBidPayload

Used for bid price adjustments.

```protobuf
message AdjustBidPayload {
  double price = 1;
}
```

##### MetricsPayload

Used for adding impression metrics.

```protobuf
message MetricsPayload {
  repeated com.iabtechlab.openrtb.v2.BidRequest.Metric metric = 1;
}
```

##### DataPayload

Used for content data (`ADD_CIDS`) and for Agentic Audiences segments with embeddings (`ADD_EMBEDDINGS`) via the `data` field.

For `ADD_EMBEDDINGS`, each `data.segment` **must** include `ext.aa` (OpenRTB Agentic Audiences envelope; see [openrtb2.x#188](https://github.com/InteractiveAdvertisingBureau/openrtb2.x/pull/188)).

```protobuf
message DataPayload {
  repeated com.iabtechlab.openrtb.v2.BidRequest.Data data = 1;
}
```

See [ADD_EMBEDDINGS](#add_embeddings) for intent-specific usage.

---

### Intents and Operations

#### Intent Enum

| Value | Name                 | Description                                           |
|-------|----------------------|-------------------------------------------------------|
| 0     | `INTENT_UNSPECIFIED` | Unspecified (invalid)                                 |
| 1     | `ACTIVATE_SEGMENTS`  | Activate user segments by external segment IDs        |
| 2     | `ACTIVATE_DEALS`     | Activate deals by external deal IDs                   |
| 3     | `SUPPRESS_DEALS`     | Suppress deals by external deal IDs                   |
| 4     | `ADJUST_DEAL_FLOOR`  | Adjust the bid floor of a specific deal               |
| 5     | `ADJUST_DEAL_MARGIN` | Adjust the deal margin of a specific deal             |
| 6     | `BID_SHADE`          | Adjust the bid price of a specific bid                |
| 7     | `ADD_METRICS`        | Add metrics to an impression                          |
| 8     | `ADD_CIDS`           | Add extended content IDs                              |
| 9     | `ADD_EMBEDDINGS`     | Add Agentic Audiences data segment(s) with embeddings |

#### Operation Enum

| Value | Name | Description |
|-------|------|-------------|
| 0 | `OPERATION_UNSPECIFIED` | Unspecified (invalid) |
| 1 | `OPERATION_ADD` | Add new data to the target |
| 2 | `OPERATION_REMOVE` | Remove data from the target |
| 3 | `OPERATION_REPLACE` | Replace existing data at the target |

#### Intent-Payload Mapping

| Intent               | Expected Payload  | Path Example                  |
|----------------------|-------------------|-------------------------------|
| `ACTIVATE_SEGMENTS`  | IDsPayload        | `/user/data/segment`          |
| `ACTIVATE_DEALS`     | IDsPayload        | `/imp/{id}`                   |
| `SUPPRESS_DEALS`     | IDsPayload        | `/imp/{id}`                   |
| `ADJUST_DEAL_FLOOR`  | AdjustDealPayload | `/imp/{id}/deals/{dealId}`    |
| `ADJUST_DEAL_MARGIN` | AdjustDealPayload | `/imp/{id}/deals/{dealId}`    |
| `BID_SHADE`          | AdjustBidPayload  | `/seatbid/{seat}/bid/{bidId}` |
| `ADD_METRICS`        | MetricsPayload    | `/imp/{id}`                   |
| `ADD_CIDS`           | DataPayload       | `/site/content/data` (`data.ext.cids`) |
| `ADD_EMBEDDINGS`     | DataPayload       | `/user/data`                  |

Detailed guides for each intent are in [Intents](#intents).

---

### Server Endpoints

#### gRPC Service

| Port | Service | Method | Description |
|------|---------|--------|-------------|
| 50051 | RTBExtensionPoint | GetMutations | Process bid request and return mutations |

#### Health Check HTTP Endpoints

| Port | Path | Method | Description |
|------|------|--------|-------------|
| 8080 | `/health/live` | GET | Liveness probe - returns 200 if process is alive |
| 8080 | `/health/ready` | GET | Readiness probe - returns 200 if ready for traffic |

#### Health Response Format

```json
{
  "status": "ready",
  "ready": true,
  "version": "1.0.0"
}
```

---

### Container Requirements

#### Security Requirements (per ARTF spec)

| Requirement | Implementation |
|-------------|----------------|
| Non-root user | `USER 65534:65534` in Dockerfile |
| Drop capabilities | `cap_drop: ALL` in docker-compose |
| Read-only filesystem | `read_only: true` in docker-compose |
| No privilege escalation | `no-new-privileges: true` |
| Network isolation | Configurable via network policies |

#### Resource Defaults

| Resource | Default | Description |
|----------|---------|-------------|
| CPU Limit | 500m | 0.5 CPU cores |
| Memory Limit | 256Mi | 256 MB RAM |
| CPU Request | 250m | 0.25 CPU cores |
| Memory Request | 128Mi | 128 MB RAM |

#### Health Probes

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

### Build and Deployment

#### Prerequisites

- Go 1.22+
- Protocol Buffers compiler (`protoc`)
- protoc-gen-go and protoc-gen-go-grpc plugins
- Docker (for containerized deployment)

#### Build Commands

| Command | Description |
|---------|-------------|
| `make deps` | Download Go dependencies |
| `make generate` | Generate protobuf Go code |
| `make build` | Build server binary |
| `make test` | Run unit tests |
| `make test-coverage` | Run tests with coverage report |
| `make lint` | Run linter |
| `make clean` | Remove build artifacts |

#### Run Commands

| Command | Description |
|---------|-------------|
| `make run` | Run server locally |
| `make docker-build` | Build Docker image |
| `make docker-run` | Run Docker container |
| `make docker-compose-up` | Start with docker-compose |
| `make docker-compose-down` | Stop docker-compose services |

#### Testing Commands

| Command | Description |
|---------|-------------|
| `make grpc-test` | Test gRPC endpoint with grpcurl |
| `make health-check` | Check health endpoints with curl |

---

### Configuration

#### Command-Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-grpc-port` | 50051 | gRPC server listening port |
| `-health-port` | 8080 | Health check HTTP server port |

#### Environment Variables

| Variable | Description |
|----------|-------------|
| `GRPC_PORT` | Override gRPC port |
| `HEALTH_PORT` | Override health check port |

#### Agent Manifest (Container Label)

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
    "BID_SHADE",
    "ADD_EMBEDDINGS"
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

### Example Mutations

#### Activate Segments

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

#### Activate Deals

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

#### Adjust Deal Floor

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

#### Bid Shading

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

#### Add Embeddings

```json
{
  "intent": "ADD_EMBEDDINGS",
  "op": "OPERATION_ADD",
  "path": "/user/data",
  "content_data": {
    "data": [
      {
        "name": "data-provider",
        "segment": [
          {
            "id": "seg-ctx-001",
            "name": "descriptive-name",
            "ext": {
              "aa": {
                "ver": "1.0.0",
                "vector": "mpkZPq5HYb5SuJ4+PQrXPo/C9T7NzEy97FE4Pilcj74=",
                "dimension": 8,
                "model": "sbert-mini-ctx-001",
                "type": [2]
              }
            }
          }
        ]
      }
    ]
  }
}
```

---

### References

- [IAB Tech Lab Agentic RTB Framework v1.0](https://iabtechlab.com/standards/artf/)
- [OpenRTB v2.6 Specification](https://iabtechlab.com/standards/openrtb/)
- [IAB OpenRTB Agentic Audiences community extension](https://github.com/InteractiveAdvertisingBureau/openrtb/blob/main/extensions/community_extensions/agentic-audiences.md)
- [OpenRTB protobuf `Segment.ext.aa` (openrtb2.x#188)](https://github.com/InteractiveAdvertisingBureau/openrtb2.x/pull/188)
- [gRPC Documentation](https://grpc.io/docs/)
- [Protocol Buffers](https://protobuf.dev/)
- [OCI Container Specification](https://opencontainers.org/)

---

*Document Version: 1.1.0*
*Last Updated: September 2026*

---

## MCP Integration

This document describes the Model Context Protocol (MCP) server integration for the Agentic RTB Framework, enabling AI agents and LLMs to interact with RTB extension points through the standardized MCP protocol.

---

### Table of Contents

1. [Overview](#overview-1)
2. [Architecture](#architecture)
3. [MCP Tool Definition](#mcp-tool-definition)
4. [Transport Options](#transport-options)
5. [Web Interface](#web-interface)
6. [Sample Payloads](#sample-payloads)
7. [Configuration](#configuration-1)
8. [Usage Examples](#usage-examples)

---

### Overview

The ARTF implementation exposes the `RTBExtensionPoint` service via MCP as a tool called **"Extend RTB"**. This enables:

- **AI Agent Integration** - LLMs can invoke RTB mutations through MCP tool calls
- **Streamable HTTP** - Web-based clients can interact via HTTP streaming
- **Web UI** - Built-in web component for testing and demonstration
- **Dual Protocol Support** - Run gRPC and MCP simultaneously on different ports

#### Why MCP?

Per the ARTF specification, while gRPC is mandated for service-to-service communication, MCP enables **model-to-agent orchestration** for autonomic agentic flows. MCP provides:

- Standardized tool definitions for LLM interaction
- JSON-RPC based communication (natural successor to REST)
- Support for both structured and streaming responses
- OAuth authentication support (MCP 2025-06-18+)

---

### Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        ARTF Agent                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐           │
│  │   gRPC       │  │    MCP       │  │    Web       │           │
│  │  :50051      │  │   :50052     │  │   :8081      │           │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘           │
│         │                 │                 │                    │
│         └─────────────────┼─────────────────┘                    │
│                           │                                      │
│                    ┌──────▼───────┐                              │
│                    │   Handlers   │                              │
│                    │  (Shared)    │                              │
│                    └──────────────┘                              │
│                                                                  │
│  ┌──────────────┐                                                │
│  │   Health     │                                                │
│  │   :8080      │                                                │
│  └──────────────┘                                                │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

#### Port Assignments

| Port | Protocol | Service | Description |
|------|----------|---------|-------------|
| 50051 | gRPC | RTBExtensionPoint | Primary service endpoint |
| 50052 | HTTP/SSE | MCP Streamable HTTP | MCP tool endpoint |
| 8080 | HTTP | Health Checks | Liveness/readiness probes |
| 8081 | HTTP | Web UI | Testing interface |

---

### MCP Tool Definition

#### Tool: "Extend RTB"

The MCP server exposes a single tool that wraps the `RTBExtensionPoint.GetMutations` RPC.

##### Tool Schema

```json
{
  "name": "extend_rtb",
  "description": "Process an OpenRTB bid request/response and return proposed mutations for segment activation, deal management, bid shading, and metrics",
  "inputSchema": {
    "type": "object",
    "properties": {
      "lifecycle": {
        "type": "string",
        "description": "Auction lifecycle stage",
        "enum": ["LIFECYCLE_UNSPECIFIED"],
        "default": "LIFECYCLE_UNSPECIFIED"
      },
      "id": {
        "type": "string",
        "description": "Unique request ID"
      },
      "tmax": {
        "type": "integer",
        "description": "Maximum response time in milliseconds",
        "default": 100
      },
      "bid_request": {
        "type": "object",
        "description": "OpenRTB v2.6 BidRequest object",
        "required": true
      },
      "bid_response": {
        "type": "object",
        "description": "OpenRTB v2.6 BidResponse object (optional)"
      }
    },
    "required": ["id", "bid_request"]
  }
}
```

##### Response Format

```json
{
  "content": [
    {
      "type": "text",
      "text": "{\"id\":\"req-123\",\"mutations\":[...],\"metadata\":{...}}"
    }
  ]
}
```

---

### Transport Options

#### Streamable HTTP (Recommended)

The MCP server uses Streamable HTTP transport for web compatibility:

```
POST /mcp    - Send JSON-RPC requests
GET  /mcp    - Establish SSE stream for server messages
DELETE /mcp  - Terminate session
```

##### Request Flow

1. Client sends `initialize` request via POST
2. Server returns session ID in `Mcp-Session-Id` header
3. Client includes session ID in subsequent requests
4. Server streams responses via SSE (GET connection)

#### SSE Transport (Alternative)

For clients requiring traditional SSE:

```
GET  /sse      - Establish SSE connection
POST /message  - Send JSON-RPC messages
```

---

### Web Interface

The built-in web interface provides:

- **Request Builder** - Visual form for constructing ORTB payloads
- **Sample Payloads** - Pre-built examples for common scenarios
- **Live Response** - Real-time mutation results display
- **MCP Inspector** - Debug MCP message flow

#### Endpoints

| Path | Description |
|------|-------------|
| `/` | Main web interface |
| `/api/samples` | List available sample payloads |
| `/api/samples/{name}` | Get specific sample payload |
| `/mcp` | MCP streamable HTTP endpoint |

#### Web Component

The interface uses a custom web component `<artf-tester>` that can be embedded:

```html
<script type="module" src="/static/artf-tester.js"></script>
<artf-tester mcp-endpoint="/mcp"></artf-tester>
```

---

### Sample Payloads

#### Basic Banner Request

```json
{
  "id": "sample-banner-001",
  "tmax": 100,
  "bid_request": {
    "id": "auction-123",
    "imp": [
      {
        "id": "imp-1",
        "banner": {
          "w": 300,
          "h": 250,
          "pos": 1
        },
        "bidfloor": 1.50,
        "bidfloorcur": "USD"
      }
    ],
    "site": {
      "id": "site-456",
      "domain": "example.com",
      "cat": ["IAB1"],
      "page": "https://example.com/article"
    },
    "user": {
      "id": "user-789",
      "yob": 1990,
      "gender": "M",
      "data": [
        {
          "id": "data-provider-1",
          "name": "Example DMP",
          "segment": [
            {"id": "seg-sports", "name": "Sports Enthusiast"}
          ]
        }
      ]
    },
    "device": {
      "ua": "Mozilla/5.0...",
      "ip": "192.168.1.1",
      "geo": {
        "country": "USA",
        "region": "CA"
      }
    }
  }
}
```

#### Video Request with Deals

```json
{
  "id": "sample-video-001",
  "tmax": 150,
  "bid_request": {
    "id": "auction-456",
    "imp": [
      {
        "id": "imp-1",
        "video": {
          "mimes": ["video/mp4"],
          "minduration": 15,
          "maxduration": 30,
          "w": 640,
          "h": 480
        },
        "bidfloor": 8.00,
        "pmp": {
          "private_auction": 1,
          "deals": [
            {
              "id": "deal-premium-video",
              "bidfloor": 10.00,
              "at": 1
            }
          ]
        }
      }
    ],
    "site": {
      "id": "site-789",
      "domain": "streaming.example.com",
      "cat": ["IAB1-6"]
    },
    "user": {
      "id": "user-456",
      "yob": 1985
    }
  }
}
```

#### Bid Response with Shading

```json
{
  "id": "sample-bidshade-001",
  "tmax": 100,
  "bid_request": {
    "id": "auction-789",
    "imp": [
      {
        "id": "imp-1",
        "banner": {"w": 728, "h": 90},
        "bidfloor": 2.00
      }
    ],
    "user": {"id": "user-123"}
  },
  "bid_response": {
    "id": "auction-789",
    "seatbid": [
      {
        "seat": "dsp-001",
        "bid": [
          {
            "id": "bid-abc",
            "impid": "imp-1",
            "price": 5.50,
            "adomain": ["advertiser.com"]
          }
        ]
      }
    ]
  }
}
```

---

### Configuration

#### Command-Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--enable-grpc` | true | Enable gRPC interface |
| `--enable-mcp` | false | Enable MCP interface |
| `--enable-web` | false | Enable web interface |
| `--grpc-port` | 50051 | gRPC port |
| `--mcp-port` | 50052 | MCP port |
| `--web-port` | 8081 | Web UI port |
| `--health-port` | 8080 | Health check port |

#### Environment Variables

| Variable | Description |
|----------|-------------|
| `ARTF_ENABLE_GRPC` | Enable gRPC server |
| `ARTF_ENABLE_MCP` | Enable MCP server |
| `ARTF_ENABLE_WEB` | Enable web interface |
| `ARTF_GRPC_PORT` | gRPC port |
| `ARTF_MCP_PORT` | MCP port |
| `ARTF_WEB_PORT` | Web UI port |

#### Example Configurations

**gRPC Only (Production)**
```bash
./artf-agent --enable-grpc --grpc-port=50051
```

**All Services (Development)**
```bash
./artf-agent --enable-grpc --enable-mcp --enable-web
```

**MCP Only (AI Integration)**
```bash
./artf-agent --enable-mcp --mcp-port=50052
```

---

### Usage Examples

#### MCP Client (Python)

```python
import httpx
import json

MCP_ENDPOINT = "http://localhost:50052/mcp"

# Initialize session
init_request = {
    "jsonrpc": "2.0",
    "id": 1,
    "method": "initialize",
    "params": {
        "protocolVersion": "2024-11-05",
        "capabilities": {},
        "clientInfo": {"name": "test-client", "version": "1.0"}
    }
}

response = httpx.post(MCP_ENDPOINT, json=init_request)
session_id = response.headers.get("Mcp-Session-Id")

# Call extend_rtb tool
tool_request = {
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/call",
    "params": {
        "name": "extend_rtb",
        "arguments": {
            "id": "test-001",
            "tmax": 100,
            "bid_request": {
                "id": "auction-123",
                "imp": [{"id": "imp-1", "banner": {"w": 300, "h": 250}}],
                "user": {"id": "user-789", "yob": 1990}
            }
        }
    }
}

response = httpx.post(
    MCP_ENDPOINT,
    json=tool_request,
    headers={"Mcp-Session-Id": session_id}
)
print(json.dumps(response.json(), indent=2))
```

#### cURL Example

```bash
# Initialize session
curl -X POST http://localhost:50052/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "initialize",
    "params": {
      "protocolVersion": "2024-11-05",
      "capabilities": {},
      "clientInfo": {"name": "curl", "version": "1.0"}
    }
  }' -i

# Note the Mcp-Session-Id header from response, then:

# Call tool
curl -X POST http://localhost:50052/mcp \
  -H "Content-Type: application/json" \
  -H "Mcp-Session-Id: <session-id>" \
  -d '{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/call",
    "params": {
      "name": "extend_rtb",
      "arguments": {
        "id": "test-001",
        "bid_request": {
          "id": "auction-123",
          "imp": [{"id": "imp-1", "banner": {"w": 300, "h": 250}}],
          "user": {"id": "user-789", "yob": 1990}
        }
      }
    }
  }'
```

#### Claude Desktop Configuration

Add to `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "artf": {
      "command": "/path/to/artf-agent",
      "args": ["--enable-mcp", "--mcp-port=50052"],
      "env": {}
    }
  }
}
```

---

### Default Implementation: Segment Activation

The default MCP handler implements segment activation based on user demographics from the RTB payload:

#### Logic

1. **Extract User Data** - Parse `user.yob`, `user.gender`, existing segments
2. **Determine Segments** - Apply demographic rules:
   - Age 18-24 → `demo-18-24`
   - Age 25-34 → `demo-25-34`
   - Age 35-44 → `demo-35-44`
   - Age 45+ → `demo-45-plus`
3. **Enrich Segments** - Re-activate existing segments from `user.data`
4. **Return Mutations** - Generate `ACTIVATE_SEGMENTS` mutations

#### Example Response

```json
{
  "id": "test-001",
  "mutations": [
    {
      "intent": "ACTIVATE_SEGMENTS",
      "op": "OPERATION_ADD",
      "path": "/user/data/segment",
      "ids": {
        "id": ["demo-25-34", "seg-sports"]
      }
    }
  ],
  "metadata": {
    "api_version": "1.0",
    "model_version": "v1.0.0"
  }
}
```

---

### CORS Support

The MCP interface includes full CORS (Cross-Origin Resource Sharing) support to enable web browsers to communicate with the MCP endpoint from different origins (e.g., the Web UI on port 8081 accessing MCP on port 50052).

#### CORS Headers

| Header | Value | Description |
|--------|-------|-------------|
| `Access-Control-Allow-Origin` | `*` | Allow requests from any origin |
| `Access-Control-Allow-Methods` | `GET, POST, DELETE, OPTIONS` | Allowed HTTP methods |
| `Access-Control-Allow-Headers` | `Content-Type, Mcp-Session-Id, Last-Event-ID` | Allowed request headers |
| `Access-Control-Expose-Headers` | `Mcp-Session-Id` | Headers exposed to browser |

#### Preflight Handling

The MCP server automatically handles `OPTIONS` preflight requests, returning a `200 OK` with appropriate CORS headers. This enables browsers to make cross-origin requests without additional configuration.

#### Example Cross-Origin Request

```javascript
// From Web UI on localhost:8081 to MCP on localhost:50052
fetch('http://localhost:50052/mcp', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Mcp-Session-Id': sessionId
  },
  body: JSON.stringify({
    jsonrpc: '2.0',
    id: 1,
    method: 'tools/call',
    params: { name: 'extend_rtb', arguments: {...} }
  })
});
```

---

### Security Considerations

#### MCP-Specific Security

- **Session Management** - Use stateful sessions in production
- **Authentication** - Implement OAuth for external access
- **Rate Limiting** - Apply per-session rate limits
- **Input Validation** - Validate all ORTB payloads
- **CORS Restrictions** - In production, restrict `Access-Control-Allow-Origin` to specific domains

#### Network Isolation

In containerized deployments, the MCP port should be:
- Exposed only to authorized AI systems
- Protected by network policies
- Monitored for unusual patterns

---

### References

- [Model Context Protocol Specification](https://modelcontextprotocol.io/)
- [mcp-go Library](https://github.com/mark3labs/mcp-go)
- [MCP Streamable HTTP Transport](https://spec.modelcontextprotocol.io/specification/basic/transports/#streamable-http)
- [ARTF Specification](https://iabtechlab.com/standards/artf/)

---

*Document Version: 0.10.0*
*Last Updated: November 2025*

---

## Intents

### ACTIVATE_SEGMENTS

Activates segments for the current bid request.

**Payload:** `IDsPayload` via the `ids` field

**Eligible paths:**
- `/user/data/segment` — target users.

**Payload fields:**
- `id`: external segment IDs to activate for the request.

#### Example mutation

```json
{
  "intent": "ACTIVATE_SEGMENTS",
  "op": "OPERATION_ADD",
  "path": "/user/data/segment",
  "ids": {
    "id": ["sports-fans", "premium-subscribers"]
  }
}
```

### ACTIVATE_DEALS

Activates a bid request for a set of deal IDs.

**Payload:** `IDsPayload` via the `ids` field

**Eligible paths:**
- `/imp/{id}` — targets a specific impression (`imp`) by ID.

**Payload fields:**
- `id`: external deal IDs to activate on the impression.

#### Example mutation

```json
{
  "intent": "ACTIVATE_DEALS",
  "op": "OPERATION_ADD",
  "path": "/imp/{id}",
  "ids": {
    "id": ["deal-001", "deal-002"]
  }
}
```

### SUPPRESS_DEALS

Suppresses specific deals from consideration, for example because of brand-safety, policy, pacing, or campaign constraints.

**Payload:** `IDsPayload` via the `ids` field

**Eligible paths:**
- `/imp/{id}` — targets a specific impression by ID.

**Payload fields:**
- `id`: external deal IDs to suppress from eligibility.

#### Example mutation

```json
{
  "intent": "SUPPRESS_DEALS",
  "op": "OPERATION_REMOVE",
  "path": "/imp/{id}",
  "ids": {
    "id": ["deal-archived", "deal-blocked"]
  }
}
```

### ADJUST_DEAL_FLOOR

Raises or lowers the floor on a specific deal while keeping the rest of the deal targeting intact.

**Payload:** `AdjustDealPayload` via the `adjust_deal` field

**Eligible paths:**
- `/imp/{id}/deals/{dealId}` — targets a specific deal within a specific impression.

**Payload fields:**
- `bidfloor`: new floor value to apply to the deal.

#### Example mutation

```json
{
  "intent": "ADJUST_DEAL_FLOOR",
  "op": "OPERATION_REPLACE",
  "path": "/imp/{id}/deals/{dealId}",
  "adjust_deal": {
    "bidfloor": 5.5
  }
}
```

### ADJUST_DEAL_MARGIN

Adjusts the margin used for a deal.

**Payload:** `AdjustDealPayload` via the `adjust_deal` field

**Eligible paths:**
- `/imp/{id}/deals/{dealId}` — targets a specific deal within a specific impression.

**Payload fields:**
- `margin`: margin adjustment object for the deal.
- `margin.value`: the amount of margin to apply.
- `margin.calculation_type`: `CPM` for absolute margin or `PERCENT` for relative margin.

#### Example mutation

```json
{
  "intent": "ADJUST_DEAL_MARGIN",
  "op": "OPERATION_REPLACE",
  "path": "/imp/{id}/deals/{dealId}",
  "adjust_deal": {
    "margin": {
      "value": 0.15,
      "calculation_type": "PERCENT"
    }
  }
}
```

### BID_SHADE

Reduces the bid price for a bid response.

**Payload:** `AdjustBidPayload` via the `adjust_bid` field

**Eligible paths:**
- `/seatbid/{seat}/bid/{bidId}` — targets a specific bid within a specific seat bid.

**Payload fields:**
- `price`: bid price to write into the bid response.

**Restrictions:**
- The adjusted price must be lower than the original bid.
- This intent only applies to a bid response value that already exists in the auction.

#### Example mutation

```json
{
  "intent": "BID_SHADE",
  "op": "OPERATION_REPLACE",
  "path": "/seatbid/{seat}/bid/{bidId}",
  "adjust_bid": {
    "price": 4.25
  }
}
```

### ADD_METRICS

Attaches additional OpenRTB metrics to an impression, such as viewability or audibility.

**Payload:** `MetricsPayload` via the `metrics` field

**Eligible paths:**
- `/imp/{id}` — targets a specific impression by ID.

**Payload fields:**
- `metric`: list of OpenRTB metric objects to attach to the impression.
- `metric.type`: metric name, such as `viewability`.
- `metric.value`: metric value, typically a 0.0 to 1.0 score.
- `metric.vendor`: source of the metric value.

#### Example mutation

```json
{
  "intent": "ADD_METRICS",
  "op": "OPERATION_ADD",
  "path": "/imp/{id}",
  "metrics": {
    "metric": [
      {
        "type": "viewability",
        "value": 0.93,
        "vendor": "EXCHANGE"
      },
      {
        "type": "audibility",
        "value": 0.81,
        "vendor": "EXCHANGE"
      }
    ]
  }
}
```

### ADD_CIDS

Attaches contextual segments containing extended content IDs.

**Payload:** `DataPayload` via the `content_data` field. Each `data` object must include `cids`.

**Eligible paths:**
- `/site/content/data` — targets site.
- `/app/content/data` — targets app. 

**Payload fields:**
- `data`: list of OpenRTB data objects to attach to content.
- `data.id`: data provider identifier.
- `data.name`: data provider name.
- `data.segment`: content identifiers or other data segments from that provider.
- `data.ext.cids`: content IDs for that content source. 

#### Example mutation

```json
{
  "intent": "ADD_CIDS",
  "op": "OPERATION_ADD",
  "path": "/site/content/data",
  "content_data": {
    "data": [
      {
        "id": "provider-1",
        "name": "Content Provider",
        "ext": {
          "cids": ["content-abc-001", "content-abc-002"]
        }
      }
    ]
  }
}
```

### ADD_EMBEDDINGS

Attaches user data segments carrying Agentic Audiences embeddings.

**Payload:** `DataPayload` via the `data` field. Each segment **must** include `ext.aa`.

**Eligible paths:**
- `/user/data` — targets the user.

**Payload fields:**
- `data`: list of OpenRTB Data objects to attach to the user
- `data.name`: data provider name
- `data.segment`: segments from the provider
- `data.segment.id`: segment identifier
- `data.segment.name`: segment name
- `data.segment.ext.aa`: **required** Agentic Audiences embedding envelope
  - `ver` — embedding schema/spec version
  - `vector` — base64 Float32 little-endian (RFC 4648)
  - `dimension` — number of Float32 values
  - `model` — producing model identifier
  - `type` — signal types: `1`=identity, `2`=contextual, `3`=reinforcement

**Dependency:** OpenRTB protobuf `Segment.ext.aa` — [openrtb2.x#188](https://github.com/InteractiveAdvertisingBureau/openrtb2.x/pull/188).

#### Example mutation

```json
{
  "intent": "ADD_EMBEDDINGS",
  "op": "OPERATION_ADD",
  "path": "/user/data",
  "content_data": {
    "data": [
      {
        "name": "data-provider",
        "segment": [
          {
            "id": "seg-ctx-001",
            "name": "descriptive-name",
            "ext": {
              "aa": {
                "ver": "1.0.0",
                "vector": "mpkZPq5HYb5SuJ4+PQrXPo/C9T7NzEy97FE4Pilcj74=",
                "dimension": 8,
                "model": "sbert-mini-ctx-001",
                "type": [2]
              }
            }
          }
        ]
      }
    ]
  }
}
```


---

## JSON Examples

Sample ORTB payloads for testing. The Rust tests in [`examples/rust/src/sample_tests.rs`](examples/rust/src/sample_tests.rs) load these files directly.

- [banner-basic.json](samples/banner-basic.json)
- [bid-shading.json](samples/bid-shading.json)
- [multi-impression.json](samples/multi-impression.json)
- [native-ad.json](samples/native-ad.json)
- [video-deals.json](samples/video-deals.json)


---

## Scripts

Build and utility scripts.

- [generate.sh](scripts/generate.sh) — generates protobuf bindings for one or more target languages (run via `make generate`).
