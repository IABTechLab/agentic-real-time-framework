# Agentic RTB Framework Makefile

BINARY=artf-agent
IMAGE ?= artf-agent
LANGUAGES=go # cpp go csharp objc python ruby js
RUST_BINARY=rust/target/release/agentic-rtb-framework-service
GRPC_ADDR ?= localhost:50051
HEALTH_URL ?= http://localhost:8080
SAMPLE_SERVICE := com.iabtechlab.bidstream.mutation.services.v1.RTBExtensionPoint/GetMutations

.DEFAULT_GOAL := build

.PHONY: help deps generate build run run-dev run-all run-grpc run-mcp run-web \
	test test-coverage lint \
	build-rust run-rust build-all \
	docker-build docker-run docker-run-all docker-compose-up docker-compose-down \
	health-check grpc-test sample-banner sample-video sample-bidshade \
	bindings check clean docs watch

help:
	@echo "Reference agents (checked-in pkg/pb/; protoc not required):"
	@echo "  make deps          # go mod download"
	@echo "  make build         # Go agent -> ./$(BINARY)"
	@echo "  make build-rust    # Rust reference service"
	@echo "  make test          # go test ./..."
	@echo "  make docker-build  # container image $(IMAGE)"
	@echo ""
	@echo "Regenerate protobufs (requires protoc; not needed to build the agents):"
	@echo "  make generate      # Go: scripts/generate.sh"
	@echo "  make bindings      # spec language bindings (repo-root protos)"
	@echo ""
	@echo "Run (requires a built binary):"
	@echo "  make run-all       # gRPC + MCP + web + health"
	@echo "  make health-check  # curl $(HEALTH_URL)/health/{live,ready}"
	@echo "  make grpc-test     # grpcurl sample against $(GRPC_ADDR)"

# Go build and run targets

deps:
	go mod download

# Go protobuf regen. Not required to build; pkg/pb/ is checked in.
generate:
	scripts/generate.sh

build:
	go build -o $(BINARY) ./cmd/agent

run: run-all

run-dev: run-all

run-all: build
	./$(BINARY) --enable-grpc --enable-mcp --enable-web

run-grpc: build
	./$(BINARY) --enable-grpc

run-mcp: build
	./$(BINARY) --enable-mcp

run-web: build
	./$(BINARY) --enable-mcp --enable-web

test:
	go test ./...

test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out

lint:
	go vet ./...

# Rust build and run targets

build-rust:
	cd rust && cargo build --release

run-rust: build-rust
	ARTF_GRPC_SERVER_PORT=50053 ARTF_HTTP_SERVER_PORT=8082 $(RUST_BINARY)

build-all: build build-rust

# Docker

docker-build:
	docker build -t $(IMAGE) .

docker-run: docker-run-all

docker-run-all: docker-build
	docker run --rm -p 50051:50051 -p 8081:8081 -p 8080:8080 $(IMAGE)

docker-compose-up:
	docker compose up --build -d

docker-compose-down:
	docker compose down

# Live checks (server must already be running)

health-check:
	curl -fsS $(HEALTH_URL)/health/live
	@echo
	curl -fsS $(HEALTH_URL)/health/ready
	@echo

grpc-test: sample-banner

sample-banner:
	grpcurl -plaintext -d @ $(GRPC_ADDR) $(SAMPLE_SERVICE) < samples/banner-basic.json

sample-video:
	grpcurl -plaintext -d @ $(GRPC_ADDR) $(SAMPLE_SERVICE) < samples/video-deals.json

sample-bidshade:
	grpcurl -plaintext -d @ $(GRPC_ADDR) $(SAMPLE_SERVICE) < samples/bid-shading.json

# Protobuf targets

bindings:
	for x in ${LANGUAGES}; do \
		protoc --proto_path=. \
			--$${x}_out=. \
			--experimental_editions \
			openrtb.proto agenticrtbframework.proto; \
		protoc --proto_path=. \
			--$${x}_out=. \
			--$${x}-grpc_out=require_unimplemented_servers=false:. \
			agenticrtbframeworkservices.proto; \
	done

check:
	prototool lint

clean:
	for x in ${LANGUAGES}; do \
		rm -fr $${x}/*; \
	done
	rm -f $(BINARY) coverage.out coverage.html

docs:
	podman run --rm \
		-v ${PWD}:${PWD} \
		-w ${PWD} \
		pseudomuto/protoc-gen-doc \
		--doc_opt=html,doc.html \
		--proto_path=${PWD} \
		openrtb.proto agenticrtbframework.proto agenticrtbframeworkservices.proto

watch:
	fswatch  -r ./ | xargs -n1 make docs
