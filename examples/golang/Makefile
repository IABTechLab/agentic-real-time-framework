# Agent service build and run targets

BINARY=artf-agent

.PHONY: build run-all run-grpc run-mcp run-web test docker-build docker-run-all

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

docker-build:
	docker build -f Dockerfile -t artf-agent:latest ../..

docker-run-all: docker-build
	docker run --rm -p 50051:50051 -p 50052:50052 -p 8081:8081 -p 8080:8080 artf-agent:latest --enable-grpc --enable-mcp --enable-web
