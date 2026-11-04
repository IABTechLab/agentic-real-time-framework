# Go Example Service

This directory contains the Go implementation for the Agentic RTB Framework (ARTF) example service.

It provides a working gRPC agent with the same mutation handlers, MCP support, and web UI pattern used by the framework reference implementation.

## Build

From the repo root:

```bash
go build ./examples/golang/cmd/agent
```

Or from inside this directory:

```bash
cd examples/golang
go build ./cmd/agent
```

## Run

```bash
cd examples/golang
go run ./cmd/agent --enable-grpc --enable-mcp --enable-web
```

## Docker

```bash
cd examples/golang
docker compose up --build
```
