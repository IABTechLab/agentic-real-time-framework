# Agentic Real Time Framework
#
# You do not need protoc to build the reference implementations.
# Generated Go lives in pkg/pb/. The Rust service compiles protos in build.rs.
# `make bindings` / `make generate` regenerate Go from proto/ and are optional.

BINARY ?= artf-agent
IMAGE ?= artf-agent
RUST_DIR := rust
RUST_BINARY := $(RUST_DIR)/target/release/agentic-rtb-framework-service
PROTO_DIR := proto
OPENRTB_PROTO := $(PROTO_DIR)/com/iabtechlab/openrtb/v2/openrtb.proto
ARTF_PROTO := $(PROTO_DIR)/agenticrtbframework.proto
GRPC_ADDR ?= localhost:50051
HEALTH_URL ?= http://localhost:8080
SAMPLE_SERVICE := com.iabtechlab.bidstream.mutation.services.v1.RTBExtensionPoint/GetMutations

.DEFAULT_GOAL := build

.PHONY: help deps fetch-openrtb generate bindings build run run-dev run-all \
	run-grpc run-mcp run-web test test-coverage lint clean \
	build-rust run-rust build-all \
	docker-build docker-run docker-run-all docker-compose-up docker-compose-down \
	health-check grpc-test sample-banner sample-video sample-bidshade \
	check docs watch

help:
	@echo "Correct way of building ARTF (issue #17):"
	@echo "  make deps          # go mod download"
	@echo "  make build         # Go agent -> ./$(BINARY)"
	@echo "  make build-rust    # Rust reference service"
	@echo "  make test          # go test ./..."
	@echo "  make docker-build  # container image $(IMAGE)"
	@echo ""
	@echo "Protobuf Go is checked in under pkg/pb/. Regeneration is optional:"
	@echo "  make generate      # confirm vendored proto + pkg/pb/"
	@echo "  make bindings      # same; proto lives under $(PROTO_DIR)/, not repo root"
	@echo ""
	@echo "Run (requires a built binary):"
	@echo "  make run-all       # gRPC + MCP + web + health"
	@echo "  make health-check  # curl $(HEALTH_URL)/health/{live,ready}"
	@echo "  make grpc-test     # grpcurl sample against $(GRPC_ADDR)"

# --- Go ---

deps:
	go mod download

# Proto is vendored. This target exists because scripts/generate.sh tells
# people to run it; it is a no-op when the file is already present.
fetch-openrtb:
	@test -f "$(OPENRTB_PROTO)" || { \
		echo "OpenRTB proto not found at $(OPENRTB_PROTO)"; \
		echo "Expected vendored file proto/com/iabtechlab/openrtb/v2/openrtb.proto"; \
		exit 1; \
	}
	@echo "OpenRTB proto vendored at $(OPENRTB_PROTO)"

generate: fetch-openrtb
	@test -f pkg/pb/artf/agenticrtbframework.pb.go
	@echo "Protobuf Go is checked in under pkg/pb/. Skipping regeneration (not required to build)."
	@echo "To regenerate from proto/: ./scripts/generate.sh"

# Historical target. It used to invoke protoc on a repo-root openrtb.proto that
# does not exist. Point at the vendored tree; do not require a regen to build.
bindings: fetch-openrtb
	@test -f "$(OPENRTB_PROTO)"
	@test -f "$(ARTF_PROTO)"
	@echo "Vendored proto sources:"
	@echo "  $(OPENRTB_PROTO)"
	@echo "  $(ARTF_PROTO)"
	@echo "Checked-in Go bindings: pkg/pb/"
	@echo "There is no openrtb.proto at the repo root. Regeneration: ./scripts/generate.sh"

build:
	go build -o $(BINARY) ./cmd/agent

run: run-all

run-dev: build
	./$(BINARY) --enable-grpc --enable-mcp --enable-web

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

clean:
	rm -f $(BINARY) coverage.out coverage.html
	rm -rf $(RUST_DIR)/target

# --- Rust ---

build-rust:
	cd $(RUST_DIR) && cargo build --release

run-rust: build-rust
	ARTF_GRPC_SERVER_PORT=50053 ARTF_HTTP_SERVER_PORT=8082 $(RUST_BINARY)

build-all: build build-rust

# --- Docker ---

docker-build:
	docker build -t $(IMAGE) .

docker-run: docker-run-all

docker-run-all: docker-build
	docker run --rm -p 50051:50051 -p 8081:8081 -p 8080:8080 $(IMAGE)

docker-compose-up:
	docker compose up --build -d

docker-compose-down:
	docker compose down

# --- Live checks (server must already be running) ---

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

# Spec leftovers. prototool is not required to build.
check:
	@echo "prototool is not part of the reference build. Use: make test"

docs:
	podman run --rm \
		-v ${PWD}:${PWD} \
		-w ${PWD} \
		pseudomuto/protoc-gen-doc \
		--doc_opt=html,doc.html \
		--proto_path=${PWD}/$(PROTO_DIR) \
		com/iabtechlab/openrtb/v2/openrtb.proto agenticrtbframework.proto

watch:
	fswatch -r ./ | xargs -n1 make docs
