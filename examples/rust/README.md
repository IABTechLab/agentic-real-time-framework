# Rust Example Service

This directory contains the Rust reference implementation example for the Agentic RTB Framework (ARTF).

It is a working example of a Rust gRPC service that follows the same contract as the Go implementation:

```proto
service RTBExtensionPoint {
  rpc GetMutations (RTBRequest) returns (RTBResponse);
}
```

## What this example does

The example service:

- generates Rust bindings from the proto definitions at build time
- exposes a gRPC `RTBExtensionPoint` server
- evaluates inbound requests and returns `RTBResponse` values
- demonstrates how a Rust service can integrate with the ARTF mutation model

## Build

From the repo root:

```bash
cargo build --manifest-path Cargo.toml
```

## Run

```bash
cargo run --manifest-path Cargo.toml
```

The example starts its gRPC server and an HTTP server on the configured ports.

## Proto source

This example uses proto files under `examples/rust/proto/`:

- `agenticrtbframework.proto`
- `agenticrtbframeworkservices.proto`

Those protos are written in `proto3` syntax and import the OpenRTB definitions:

```proto
import "com/iabtechlab/openrtb/v2.6/openrtb.proto";
```

The request and response messages include fields such as:

- `RTBRequest`
- `RTBResponse`
- `Mutation`
- `Metadata`
- `BidRequest` / `BidResponse` from OpenRTB

## Important caveat

This example is intentionally using a proto3-based compatibility layer that references the OpenRTB schema, rather than directly compiling the repo’s newest canonical proto declarations.

The repo’s top-level proto definitions use newer proto syntax (`edition = "2023"`). The Rust toolchain in this environment does not fully parse that syntax through the installed `prost`/`tonic` stack, so the example project uses the compatible `proto3` arrangement under `examples/rust/proto/`.

This keeps the Rust example buildable while preserving the same ARTF service contract and OpenRTB shape.