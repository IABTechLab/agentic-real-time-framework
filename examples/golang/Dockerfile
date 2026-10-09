# Build stage
FROM golang:1.23 AS builder

WORKDIR /app

# Copy workspace and project source needed for the multi-module setup.
COPY go.work ./
COPY pkg ./pkg
COPY examples/golang ./examples/golang

WORKDIR /app/examples/golang

# Download dependencies
RUN go mod download

# Build arguments for versioning
ARG VERSION=0.10.0

# Build the binary with version info
RUN CGO_ENABLED=0 GOOS=linux go build \
   -ldflags="-w -s -X main.Version=${VERSION}" \
   -o /artf-agent \
   ./cmd/agent

# Runtime stage
FROM ubuntu:24.04

ARG VERSION=0.10.0
ARG AGENT_NAME=artf-reference-agent
ARG AGENT_VENDOR="IAB Tech Lab"
ARG AGENT_OWNER=artf@iabtechlab.com

LABEL agent-manifest="{ \
 \"name\": \"${AGENT_NAME}\", \
 \"version\": \"${VERSION}\", \
 \"vendor\": \"${AGENT_VENDOR}\", \
 \"owner\": \"${AGENT_OWNER}\", \
 \"resources\": { \
   \"cpu\": \"500m\", \
   \"memory\": \"256Mi\" \
 }, \
 \"intents\": [ \
   \"ACTIVATE_SEGMENTS\", \
   \"ACTIVATE_DEALS\", \
   \"SUPPRESS_DEALS\", \
   \"ADJUST_DEAL_FLOOR\", \
   \"ADJUST_DEAL_MARGIN\", \
   \"BID_SHADE\", \
   \"ADD_METRICS\" \
 ], \
 \"health\": { \
   \"livenessProbe\": { \
     \"httpGet\": { \"path\": \"/health/live\", \"port\": 8080 } \
   }, \
   \"readinessProbe\": { \
     \"httpGet\": { \"path\": \"/health/ready\", \"port\": 8080 } \
   } \
 } \
}"
LABEL org.opencontainers.image.title="${AGENT_NAME}"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.vendor="${AGENT_VENDOR}"
LABEL org.opencontainers.image.description="ARTF Reference Agent - Agentic RTB Framework implementation"
LABEL org.opencontainers.image.source="https://github.com/IABTechLab/agentic-rtb-framework"
LABEL org.opencontainers.image.licenses="AGPL-3.0"

RUN apt-get update && apt-get install -y --no-install-recommends \
   ca-certificates \
   tzdata \
   && rm -rf /var/lib/apt/lists/*

COPY --from=builder /artf-agent /artf-agent

USER nobody

EXPOSE 50051 8081 8080

HEALTHCHECK --interval=5s --timeout=3s --start-period=5s --retries=3 \
   CMD ["/artf-agent", "-health-check"] || exit 1

ENTRYPOINT ["/artf-agent"]
CMD ["--enable-grpc", "--enable-mcp", "--enable-web", "--grpc-port=50051", "--web-port=8081", "--health-port=8080"]
