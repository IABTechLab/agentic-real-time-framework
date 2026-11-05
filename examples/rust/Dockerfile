
FROM rust:slim

LABEL agent-manifest='{ \
        "name": "agentic-rtb-framework", \
        "version": "1.0.0", \
        "vendor": "reference-vendor", \
        "owner": "owner@example.com", \
        "resources": {    \
            "cpu": "500m", \
            "memory": "256Mi" \
        }, \
        "intents"=["ACTIVATE_SEGMENTS", "ACTIVATE_DEALS", "BID_SHADE"], \
        "description": "Agentic RTB Framework Service"  \
        "dependencies": { \
            "audienceService": {"service": "audience-svc",  "port": 50051}   \
        },  \
        "health": {   \
            "livenessProbe": {    \
                "httpGet": { "path": "/health/live", "port": 8080 } \
            },  \
            "readinessProbe": {   \
                "httpGet": { "path": "/health/ready", "port": 8080 }    \
            }   \
        } \
    }'

WORKDIR /usr/src/agenticrtbframework
ENV BASE_DIR=/usr/src/agenticrtbframework
ENV DATA_DIR=${BASE_DIR}/data

RUN mkdir -p ${BASE_DIR}/proto

COPY proto/*.proto ${BASE_DIR}/proto/
COPY src/*.rs ./src/
COPY cargo.toml ./Cargo.toml

RUN cargo install --path .
RUN apt-get update && apt-get install -y protobuf-compiler

ENV OUT_DIR=${BASE_DIR}/proto
ENV ARTF_GRPC_SERVER_PORT=50051
ENV ARTF_GRPC_MAX_CONNS=256
ENV ARTF_HTTP_SERVER_PORT=8080

EXPOSE ${ARTF_GRPC_SERVER_PORT} ${ARTF_HEALTH_SERVER_PORT}

ENTRYPOINT ["agentic-rtb-framework-service"]