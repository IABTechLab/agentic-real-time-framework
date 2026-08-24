#!/bin/bash
# Generate Go code from protobuf definitions
# Requires: protoc, protoc-gen-go, protoc-gen-go-grpc
#
# OpenRTB 2.6 proto is fetched from IAB Tech Lab repository:
# https://github.com/IABTechLab/openrtb-proto-v2

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
PROTO_DIR="$PROJECT_ROOT/proto"
OUT_DIR="$PROJECT_ROOT/pkg/pb"

# OpenRTB proto location (downloaded by make fetch-openrtb)
OPENRTB_PROTO="$PROTO_DIR/com/iabtechlab/openrtb/v2/openrtb.proto"

# Check if OpenRTB proto exists
if [ ! -f "$OPENRTB_PROTO" ]; then
  echo "Error: OpenRTB proto not found at $OPENRTB_PROTO"
  echo "Run 'make fetch-openrtb' to download it from IAB Tech Lab repository"
  exit 1
fi

# Ensure output directories exist
mkdir -p "$OUT_DIR/openrtb"
mkdir -p "$OUT_DIR/artf"

echo "Generating Go code from protobuf definitions..."

# Generate OpenRTB 2.6 types (from IAB Tech Lab openrtb-proto-v2)
echo "  - OpenRTB 2.6 types..."
protoc \
  --proto_path="$PROTO_DIR" \
  --go_out="$PROJECT_ROOT" \
  --go_opt=module=github.com/iabtechlab/agentic-rtb-framework \
  "$OPENRTB_PROTO"

# Generate ARTF types
echo "  - ARTF types..."
protoc \
  --proto_path="$PROTO_DIR" \
  --go_out="$PROJECT_ROOT" \
  --go_opt=module=github.com/iabtechlab/agentic-rtb-framework \
  "$PROTO_DIR/agenticrtbframework.proto"

# Generate ARTF gRPC service
echo "  - ARTF gRPC service..."
protoc \
  --proto_path="$PROTO_DIR" \
  --proto_path="$PROJECT_ROOT" \
  --go_out="$PROJECT_ROOT" \
  --go_opt=module=github.com/iabtechlab/agentic-rtb-framework \
  --go-grpc_out="$PROJECT_ROOT" \
  --go-grpc_opt=module=github.com/iabtechlab/agentic-rtb-framework \
  "$PROJECT_ROOT/agenticrtbframeworkservices.proto"

echo "Done! Generated files in $OUT_DIR"
