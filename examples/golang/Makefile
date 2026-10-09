# Agent service build and run targets

BINARY=artf-agent

.PHONY: deps build run-all run-grpc run-mcp run-web test test-coverage lint docker-build docker-run-all health-check grpc-test sample-banner sample-video sample-bidshade

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

deps:
	go mod download

test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out

lint:
	go vet ./...

GRPC_ADDR ?= localhost:50051
HEALTH_URL ?= http://localhost:8080
SAMPLE_SERVICE := com.iabtechlab.bidstream.mutation.services.v1.RTBExtensionPoint/GetMutations

health-check:
	curl -fsS $(HEALTH_URL)/health/live
	@echo
	curl -fsS $(HEALTH_URL)/health/ready
	@echo

grpc-test: sample-banner

sample-banner:
	grpcurl -plaintext -d @ $(GRPC_ADDR) $(SAMPLE_SERVICE) < ../../samples/banner-basic.json

sample-video:
	grpcurl -plaintext -d @ $(GRPC_ADDR) $(SAMPLE_SERVICE) < ../../samples/video-deals.json

sample-bidshade:
	grpcurl -plaintext -d @ $(GRPC_ADDR) $(SAMPLE_SERVICE) < ../../samples/bid-shading.json
