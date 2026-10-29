# Build stage
FROM rust:slim-trixie AS builder

# build.rs compiles the protos at build time and needs protoc.
RUN apt-get update && apt-get install -y --no-install-recommends protobuf-compiler \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /usr/src/agenticrtbframework

COPY Cargo.toml Cargo.lock build.rs ./
COPY proto ./proto
COPY src ./src

RUN cargo build --release --locked

# Runtime stage
FROM debian:trixie-slim

LABEL agent-manifest='{ \
        "name": "agentic-rtb-framework", \
        "version": "1.0.0", \
        "vendor": "reference-vendor", \
        "owner": "owner@example.com", \
        "description": "Agentic RTB Framework Service", \
        "resources": { \
            "cpu": "500m", \
            "memory": "256Mi" \
        }, \
        "intents": ["ACTIVATE_SEGMENTS", "ACTIVATE_DEALS", "BID_SHADE"], \
        "health": { \
            "livenessProbe": { \
                "httpGet": { "path": "/health/live", "port": 8080 } \
            }, \
            "readinessProbe": { \
                "httpGet": { "path": "/health/ready", "port": 8080 } \
            } \
        } \
    }'

COPY --from=builder /usr/src/agenticrtbframework/target/release/agentic-rtb-framework-service /usr/local/bin/agentic-rtb-framework-service

ENV ARTF_GRPC_SERVER_PORT=50051
ENV ARTF_HTTP_SERVER_PORT=8080
ENV ARTF_MAX_CONNS=256

USER nobody

EXPOSE 50051 8080

ENTRYPOINT ["agentic-rtb-framework-service"]
