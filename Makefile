# Agentic RTB Framework Makefile

BINARY=artf-agent
LANGUAGES=go # cpp go csharp objc python ruby js

# Go build and run targets
.PHONY: build run-all run-grpc run-mcp run-web test

build:
	go build -o $(BINARY) ./cmd/agent

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

# Rust build and run targets
RUST_BINARY=rust/target/release/agentic-rtb-framework-service

.PHONY: build-rust run-rust build-all

build-rust:
	cd rust && cargo build --release

run-rust: build-rust
	ARTF_GRPC_SERVER_PORT=50053 ARTF_HTTP_SERVER_PORT=8082 $(RUST_BINARY)

build-all: build build-rust

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
