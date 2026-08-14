# CLAUDE.md - Agentic RTB Framework

This repository implements an agent service based on the IAB Tech Lab's **Agentic RTB Framework (ARTF) v1.0** specification. The framework enables agent-driven containers to participate in OpenRTB bidstream processing.

**Current Version:** 0.10.0

## Overview

The Agentic RTB Framework defines a standard for implementing agent services that:
- Operate within a host platform's infrastructure
- Process bidstream data in real-time (sub-millisecond latency requirements)
- Propose mutations to OpenRTB bid requests/responses via the "OpenRTB Patch" protocol
- Communicate via gRPC with protobuf serialization
- Support MCP (Model Context Protocol) for AI agent integration

## Project Structure

```
.
├── CLAUDE.md                           # This file
├── README.md                           # Project readme
├── Makefile                            # Build automation
├── Dockerfile                          # Container build
├── cmd/agent/                          # Main entry point
├── internal/
│   ├── agent/                          # gRPC service implementation
│   ├── mcp/                            # MCP interface implementation
│   ├── handlers/                       # Mutation handlers
│   ├── health/                         # Health check endpoints
│   └── web/                            # Web UI
├── pkg/pb/                             # Generated protobuf code
├── proto/                              # Protobuf definitions
├── docs/                               # Documentation
└── samples/                            # Sample request payloads
```

## Protocol Definition

### Service: RTBExtensionPoint

The core gRPC service that agents must implement:

```protobuf
service RTBExtensionPoint {
  rpc GetMutations (RTBRequest) returns (RTBResponse);
}
```

### Key Messages

| Message | Description |
|---------|-------------|
| `RTBRequest` | Contains lifecycle stage, request ID, tmax, bid_request, optional bid_response |
| `RTBResponse` | Returns request ID, list of mutations, and metadata |
| `Mutation` | Defines intent, operation (add/remove/replace), path, and value payload |

### Supported Intents

| Intent | Value | Description |
|--------|-------|-------------|
| `ACTIVATE_SEGMENTS` | 1 | Activate user segments by external segment IDs |
| `ACTIVATE_DEALS` | 2 | Activate deals by external deal IDs |
| `SUPPRESS_DEALS` | 3 | Suppress deals by external deal IDs |
| `ADJUST_DEAL_FLOOR` | 4 | Adjust the bid floor of a specific deal |
| `ADJUST_DEAL_MARGIN` | 5 | Adjust the deal margin |
| `BID_SHADE` | 6 | Adjust the bid price |
| `ADD_METRICS` | 7 | Add metrics to an impression |

### Operations

| Operation | Value | Description |
|-----------|-------|-------------|
| `OPERATION_ADD` | 1 | Add new data |
| `OPERATION_REMOVE` | 2 | Remove existing data |
| `OPERATION_REPLACE` | 3 | Replace existing data |

### Payload Types

- `IDsPayload` - List of string identifiers (for segments, deals)
- `AdjustDealPayload` - Bidfloor and margin adjustments
- `AdjustBidPayload` - Bid price adjustments
- `AddMetricsPayload` - OpenRTB Metric objects

## Development Commands

```bash
# Build the agent (includes protobuf generation)
make build

# Run with all interfaces enabled (gRPC, MCP, Web)
make run-all

# Run specific interfaces
make run-grpc    # gRPC only (port 50051)
make run-mcp     # MCP only (port 50052)
make run-web     # Web + MCP (ports 8081, 50052)

# Run tests
make test

# Build Docker image
make docker-build

# Run with Docker
docker run -p 50051:50051 -p 50052:50052 -p 8081:8081 artf-agent:latest --enable-grpc --enable-mcp --enable-web
```

### Port Configuration

| Port | Service | Description |
|------|---------|-------------|
| 50051 | gRPC | RTBExtensionPoint service |
| 50052 | MCP | Model Context Protocol (standalone mode) |
| 8080 | Health | Kubernetes health probes |
| 8081 | Web UI | Testing and demo interface (+ MCP at `/mcp` when both enabled) |

**Note:** When both `--enable-web` and `--enable-mcp` are set, MCP is served on the Web UI port (8081) at `/mcp` instead of its own port. This simplifies load balancer configuration.

### External URL / Load Balancer Support

Use `--external-url` to specify an external base URL when deploying behind a load balancer:

```bash
./artf-agent --enable-grpc --enable-mcp --enable-web \
  --external-url "https://rtb.example.com"
```

This rewrites all service URLs to use the external address, ensuring the Web UI's MCP endpoint points to the load balancer URL (e.g., `https://rtb.example.com/mcp`).

### MCP CORS Support

The MCP interface includes CORS headers to enable cross-origin requests from web browsers:
- `Access-Control-Allow-Origin: *`
- `Access-Control-Allow-Methods: GET, POST, DELETE, OPTIONS`
- `Access-Control-Allow-Headers: Content-Type, Mcp-Session-Id, Last-Event-ID`
- `Access-Control-Expose-Headers: Mcp-Session-Id`

## Implementation Requirements

### Container Requirements (from ARTF spec)

1. **Must run as non-root user**
2. **Must implement Kubernetes health probes**:
   - Liveness probe: `GET /health/live`
   - Readiness probe: `GET /health/ready`
3. **Must follow least-privilege principle** - drop unnecessary capabilities
4. **Must handle graceful shutdowns**
5. **No external network access** - only communicate with orchestrator services
6. **Must support OpenTelemetry** for metrics and distributed tracing

### Performance Requirements

- Sub-millisecond response times expected
- Use efficient languages (Go, Rust, Java recommended)
- gRPC with protobuf for serialization efficiency
- Respect `tmax` timeout from requests

### Agent Manifest

Containers must include an `agent-manifest` label in image metadata with:

```json
{
  "name": "agent-name",
  "version": "1.0.0",
  "vendor": "vendor-name",
  "owner": "owner@example.com",
  "resources": {
    "cpu": "500m",
    "memory": "256Mi"
  },
  "intents": ["ACTIVATE_SEGMENTS", "ACTIVATE_DEALS"],
  "dependencies": {
    "serviceName": {
      "service": "svc-name",
      "port": 9000
    }
  },
  "health": {
    "livenessProbe": { "httpGet": { "path": "/health/live", "port": 8080 } },
    "readinessProbe": { "httpGet": { "path": "/health/ready", "port": 8080 } }
  }
}
```

## Code Style Guidelines

- Use Go for implementation (efficient, good gRPC support)
- Follow standard Go project layout
- Use `context.Context` for cancellation and timeouts
- Implement proper error handling with gRPC status codes
- Add structured logging with OpenTelemetry integration
- Write unit tests for all mutation handlers

## Example Mutation Response

```json
{
  "id": "request-123",
  "mutations": [
    {
      "intent": "ACTIVATE_SEGMENTS",
      "op": "OPERATION_ADD",
      "path": "/user/data/segment",
      "ids": {
        "id": ["18-35-age-segment", "soccer-watchers"]
      }
    }
  ],
  "metadata": {
    "api_version": "1.0",
    "model_version": "v2.1"
  }
}
```

## Dependencies

The protobuf imports OpenRTB v2.6 definitions:
```protobuf
import "com/iabtechlab/openrtb/v2/openrtb.proto";
```

You'll need the IAB Tech Lab OpenRTB protobuf definitions from:
https://github.com/InteractiveAdvertisingBureau/openrtb

## References

- [Agentic RTB Framework Specification](https://iabtechlab.com/standards/artf/)
- [OpenRTB Specification](https://iabtechlab.com/standards/openrtb/)
- [IAB Tech Lab GitHub](https://github.com/InteractiveAdvertisingBureau)
- [gRPC Go Documentation](https://grpc.io/docs/languages/go/)
- [Protocol Buffers](https://protobuf.dev/)

## License

The Agentic RTB Framework specification is licensed under Creative Commons Attribution 3.0.
