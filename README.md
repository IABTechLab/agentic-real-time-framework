![IAB Tech Lab](https://drive.google.com/uc?id=10yoBoG5uRETSXRrnJPUDuONujvADrSG1)

# **Agentic Real Time Framework**

#### About Agentic Real Time Framework (ARTF)
https://iabtechlab.com/standards/artf/

#### How to get started

Download the openRTB official 2.6 Protocol Buffers specification from https://github.com/InteractiveAdvertisingBureau/openrtb2.x/blob/main/proto/src/main/com/iabtechlab/openrtb/v2/openrtb.proto to this directory.

From the command line:

1. Install `make` and the latest version of `protoc`.
2. Open the `Makefile` and choose the language(s) for which the Protocol Buffers
   object code should be generated.
3. Run `make`.

#### Contact
For more information, or to get involved, please email support@iabtechlab.com.

---

## Go Reference Implementation

A Go implementation of the IAB Tech Lab's **Agentic Real Time Framework (ARTF) v1.0** specification for agent-driven containers in OpenRTB and Digital Advertising.

### Overview

This project implements a multi-protocol server that conforms to the ARTF specification, enabling:

- **Segment Activation** - Activate user segments based on bid request data
- **Deal Management** - Activate, suppress, and adjust deals dynamically
- **Bid Shading** - Optimize bid prices using intelligent pricing strategies
- **Metrics Addition** - Add viewability and other metrics to impressions

### Supported Protocols

| Protocol | Port | Description |
|----------|------|-------------|
| gRPC | 50051 | Native RTBExtensionPoint service |
| MCP | 50052* | Model Context Protocol for AI agents |
| Web UI | 8081 | Browser-based testing interface |
| Health | 8080 | Kubernetes liveness/readiness probes |

*When both Web UI and MCP are enabled, MCP is served on the Web UI port (8081) at `/mcp` for simplified load balancer configuration.

### Quick Start

#### Prerequisites

- Go 1.23+
- Protocol Buffers compiler (`protoc`) v3.21+
- Docker (optional, for containerized deployment)

#### Critical Dependencies

The following tools must be installed to generate protobuf code:

| Tool | Version | Installation |
|------|---------|--------------|
| `protoc` | 3.21+ | System package manager or [releases](https://github.com/protocolbuffers/protobuf/releases) |
| `protoc-gen-go` | 1.34+ | `go install google.golang.org/protobuf/cmd/protoc-gen-go@latest` |
| `protoc-gen-go-grpc` | 1.4+ | `go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest` |

**Important:** Ensure `$(go env GOPATH)/bin` is in your `PATH`:
```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

Go module dependencies (managed via `go.mod`):

| Package | Version | Purpose |
|---------|---------|---------|
| `google.golang.org/grpc` | 1.64.0 | gRPC framework |
| `google.golang.org/protobuf` | 1.34.1 | Protocol Buffers runtime |
| `github.com/mark3labs/mcp-go` | 0.43.1 | Model Context Protocol server |

#### Build and Run

```bash
# Install dependencies
make deps

# Generate protobuf code
make generate

# Build the server
make build

# Run with all services enabled
make run-all

# Run in development mode (verbose)
make run-dev
```

#### Docker Deployment

```bash
# Build Docker image
make docker-build

# Run with Docker
make docker-run-all

# Or use docker-compose
make docker-compose-up
```

### Architecture

```
.
├── cmd/agent/           # Main agent entry point
├── internal/
│   ├── agent/           # gRPC agent implementation
│   ├── handlers/        # Mutation handlers for different intents
│   ├── health/          # Kubernetes health check endpoints
│   ├── mcp/             # MCP server implementation
│   └── web/             # Web UI for testing
├── pkg/pb/              # Generated protobuf Go code
├── proto/               # Protocol buffer definitions
│   ├── agenticrtbframework.proto  # ARTF service definition
│   └── com/iabtechlab/openrtb/    # OpenRTB v2.6 definitions
├── samples/             # Sample ORTB payloads for testing
├── docs/                # Specifications and documentation
├── scripts/             # Build and utility scripts
├── Dockerfile           # Container build definition
└── docker-compose.yml   # Local development setup
```

### API

#### RTBExtensionPoint Service (gRPC)

```protobuf
service RTBExtensionPoint {
  rpc GetMutations (RTBRequest) returns (RTBResponse);
}
```

#### MCP Tool: extend_rtb

The MCP server exposes an `extend_rtb` tool that accepts OpenRTB bid requests and returns proposed mutations.

### Supported Intents

| Intent | Description |
|--------|-------------|
| `ACTIVATE_SEGMENTS` | Activate user segments by external segment IDs |
| `ACTIVATE_DEALS` | Activate deals by external deal IDs |
| `SUPPRESS_DEALS` | Suppress deals by external deal IDs |
| `ADJUST_DEAL_FLOOR` | Adjust the bid floor of a specific deal |
| `ADJUST_DEAL_MARGIN` | Adjust the deal margin |
| `BID_SHADE` | Adjust the bid price |
| `ADD_METRICS` | Add metrics to an impression |

### Configuration

| Flag | Default | Description |
|------|---------|-------------|
| `--listen` | "" | Bind address for all services (default: all interfaces) |
| `--external-url` | "" | External base URL for load balancer (rewrites all service URLs) |
| `--enable-grpc` | true | Enable gRPC server |
| `--enable-mcp` | false | Enable MCP server |
| `--enable-web` | false | Enable web interface |
| `--grpc-port` | 50051 | gRPC server port |
| `--mcp-port` | 50052 | MCP server port (ignored when both Web and MCP enabled) |
| `--web-port` | 8081 | Web interface port |
| `--health-port` | 8080 | Health check HTTP port |

#### Load Balancer Configuration

When deploying behind a load balancer, use `--external-url` to ensure all generated URLs point to the external address:

```bash
./artf-agent --enable-grpc --enable-mcp --enable-web \
  --external-url "https://rtb.example.com"
```

This configures the agent so that:
- Web UI and MCP are served on the same port (8081)
- The MCP endpoint URL shown in the Web UI will be `https://rtb.example.com/mcp`
- All health and service URLs use the external base URL

### Testing

```bash
# Run unit tests
make test

# Run with coverage
make test-coverage

# Test gRPC endpoint (requires grpcurl)
make grpc-test

# Test MCP endpoint
make mcp-test

# Check health endpoints
make health-check

# Send sample requests via MCP
make sample-banner
make sample-video
make sample-bidshade
```

### Security

This implementation follows ARTF security requirements:

- Runs as non-root user
- Drops unnecessary capabilities
- Read-only filesystem
- No external network access (configurable)
- Health probes for Kubernetes integration

### Documentation

- [Implementation Specification](docs/00-EXAMPLE.md)
- [MCP Integration Guide](docs/01-MCP.md)

### AI Assistant Integration (MCP)

The ARTF agent can be added as an MCP server to AI assistants that support the Model Context Protocol, giving them access to the `extend_rtb` tool for processing OpenRTB bid requests.

#### Claude Desktop

Add to your Claude Desktop configuration file:

**macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`
**Windows:** `%APPDATA%\Claude\claude_desktop_config.json`
**Linux:** `~/.config/Claude/claude_desktop_config.json`

```json
{
  "mcpServers": {
    "artf": {
      "command": "/path/to/artf-agent",
      "args": ["--enable-mcp", "--enable-grpc=false"],
      "env": {}
    }
  }
}
```

After saving, restart Claude Desktop. The `extend_rtb` tool will be available for RTB mutation requests.

#### Claude Code (CLI)

Create a `.mcp.json` file in your project root or home directory:

```json
{
  "mcpServers": {
    "artf": {
      "command": "/path/to/artf-agent",
      "args": ["--enable-mcp", "--enable-grpc=false"],
      "env": {}
    }
  }
}
```

Or add to your existing Claude Code settings.

#### Remote MCP Server (HTTP)

For remote deployments, run the agent with MCP enabled and connect via HTTP:

```bash
# Start the agent with MCP over HTTP
./artf-agent --enable-mcp --enable-web --web-port=8081

# MCP endpoint will be available at: http://localhost:8081/mcp
```

For Claude Desktop with a remote server, use an MCP proxy or configure your client to connect to the HTTP endpoint.

#### ChatGPT

ChatGPT does not natively support MCP. However, you can:

1. **Use the Web UI** - Access the built-in testing interface at `http://localhost:8081`
2. **Custom GPT with Actions** - Create a Custom GPT that calls the MCP HTTP endpoint via Actions (requires exposing the endpoint publicly)
3. **API Integration** - Use the gRPC or MCP HTTP API directly in your application

#### Available MCP Tool

Once connected, the following tool is available:

| Tool | Description |
|------|-------------|
| `extend_rtb` | Process OpenRTB bid request/response and return proposed mutations |

Example prompt: *"Use extend_rtb to activate segments for a user born in 1990 viewing a sports website"*

---

#### About IAB Tech Lab
The IAB Technology Laboratory is a nonprofit research and development consortium charged
with producing and helping companies implement global industry technical standards and
solutions. The goal of the Tech Lab is to reduce friction associated with the digital advertising
and marketing supply chain while contributing to the safe growth of an industry.
The IAB Tech Lab spearheads the development of technical standards, creates and maintains a
code library to assist in rapid, cost-effective implementation of IAB standards, and establishes a
test platform for companies to evaluate the compatibility of their technology solutions with IAB
standards, which for 18 years have been the foundation for interoperability and profitable growth
in the digital advertising supply chain.

Learn more about IAB Tech Lab here: [https://www.iabtechlab.com/](https://www.iabtechlab.com/)

### License

**Reference Implementation (Source Code):** The Go reference implementation in this repository is Copyright (c) 2025 Index Exchange Inc. and is licensed under the GNU Affero General Public License v3.0 (AGPL-3.0). See the [LICENSE](LICENSE) file for details.

**ARTF Specification:** The IAB Tech Lab ARTF Specification is licensed under a Creative Commons Attribution 3.0 License. To view a copy of this license, visit creativecommons.org/licenses/by/3.0/ or write to Creative Commons, 171 Second Street, Suite 300, San Francisco, CA 94105, USA.

By submitting an idea, specification, software code, document, file, or other material (each, a "Submission") to the ARTF repository, or to the IAB Tech Lab in relation to ARTF you agree to and hereby license such Submission to the IAB Tech Lab under the Creative Commons Attribution 3.0 License and agree that such Submission may be used and made available to the public under the terms of such license.  If you are a member of the IAB Tech Lab then the terms and conditions of the [IPR Policy](https://iabtechlab.com/ipr-iab-techlab/acknowledge-ipr/) may also be applicable to your Submission, and if the IPR Policy is applicable to your Submission then the IPR Policy will control  in the event of a conflict between the Creative Commons Attribution 3.0 License and the IPR Policy.

#### Disclaimer

THE STANDARDS, THE SPECIFICATIONS, THE MEASUREMENT GUIDELINES, AND ANY OTHER MATERIALS OR SERVICES PROVIDED TO OR USED BY YOU HEREUNDER (THE "PRODUCTS AND SERVICES") ARE PROVIDED "AS IS" AND "AS AVAILABLE," AND IAB TECHNOLOGY LABORATORY, INC. ("TECH LAB") MAKES NO WARRANTY WITH RESPECT TO THE SAME AND HEREBY DISCLAIMS ANY AND ALL EXPRESS, IMPLIED, OR STATUTORY WARRANTIES, INCLUDING, WITHOUT LIMITATION, ANY WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE, AVAILABILITY, ERROR-FREE OR UNINTERRUPTED OPERATION, AND ANY WARRANTIES ARISING FROM A COURSE OF DEALING, COURSE OF PERFORMANCE, OR USAGE OF TRADE. TO THE EXTENT THAT TECH LAB MAY NOT AS A MATTER OF APPLICABLE LAW DISCLAIM ANY IMPLIED WARRANTY, THE SCOPE AND DURATION OF SUCH WARRANTY WILL BE THE MINIMUM PERMITTED UNDER SUCH LAW. THE PRODUCTS AND SERVICES DO NOT CONSTITUTE BUSINESS OR LEGAL ADVICE. TECH LAB DOES NOT WARRANT THAT THE PRODUCTS AND SERVICES PROVIDED TO OR USED BY YOU HEREUNDER SHALL CAUSE YOU AND/OR YOUR PRODUCTS OR SERVICES TO BE IN COMPLIANCE WITH ANY APPLICABLE LAWS, REGULATIONS, OR SELF-REGULATORY FRAMEWORKS, AND YOU ARE SOLELY RESPONSIBLE FOR COMPLIANCE WITH THE SAME, INCLUDING, BUT NOT LIMITED TO, DATA PROTECTION LAWS, SUCH AS THE PERSONAL INFORMATION PROTECTION AND ELECTRONIC DOCUMENTS ACT (CANADA), THE DATA PROTECTION DIRECTIVE (EU), THE E-PRIVACY DIRECTIVE (EU), THE GENERAL DATA PROTECTION REGULATION (EU), AND THE E-PRIVACY REGULATION (EU) AS AND WHEN THEY BECOME EFFECTIVE.
