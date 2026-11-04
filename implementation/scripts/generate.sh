#!/usr/bin/env bash
# Generate protobuf bindings for one or more target languages.
#
# Examples:
#   ./scripts/generate.sh --lang go
#   ./scripts/generate.sh --lang go,rust
#   ./scripts/generate.sh --lang all

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
PROTO_DIR="$PROJECT_ROOT/proto"
GO_OPENRTB_MODULE="github.com/iabtechlab/agentic-rtb-framework/pkg/pb/openrtb"
GO_ARTF_MODULE="github.com/iabtechlab/agentic-rtb-framework/pkg/pb/artf"
SUPPORTED_LANGS=(go rust cpp java)
PROTO_FILES=()
while IFS= read -r proto_file; do
  PROTO_FILES+=("$proto_file")
done < <(find "$PROTO_DIR" -type f -name '*.proto' | sort)

if [ "${#PROTO_FILES[@]}" -eq 0 ]; then
  echo "Error: no .proto files found under $PROTO_DIR"
  exit 1
fi

usage() {
  cat <<'EOF'
Usage: ./scripts/generate.sh [--lang LANG[,LANG...]]

Supported languages: go, rust, cpp, java, all

Examples:
  ./scripts/generate.sh --lang go
  ./scripts/generate.sh --lang rust
  ./scripts/generate.sh --lang cpp
  ./scripts/generate.sh --lang java
  ./scripts/generate.sh --lang go,rust
  ./scripts/generate.sh --lang all
EOF
}

if ! command -v protoc >/dev/null 2>&1; then
  echo "Error: protoc is required but not installed or not on PATH."
  exit 1
fi

LANGS=()
while [ "$#" -gt 0 ]; do
  case "$1" in
   -l|--lang|--language)
     shift
     if [ "$#" -eq 0 ]; then
       echo "Error: --lang requires a value."
       usage
       exit 1
     fi
     IFS=',' read -r -a EXTRA_LANGS <<< "$1"
     for lang in "${EXTRA_LANGS[@]}"; do
       LANGS+=("${lang//[[:space:]]/}")
     done
     ;;
   -h|--help)
     usage
     exit 0
     ;;
   *)
     echo "Error: unknown argument: $1"
     usage
     exit 1
     ;;
  esac
  shift
done

if [ "${#LANGS[@]}" -eq 0 ]; then
  LANGS=("go")
fi

get_output_dir() {
  local lang="$1"
  local target
  local idx

  for idx in "${!SUPPORTED_LANGS[@]}"; do
   target="${SUPPORTED_LANGS[$idx]}"
   if [ "$target" = "$lang" ]; then
     echo "$PROJECT_ROOT/pkg"
     return 0
   fi
  done

  echo "Error: unsupported language '$lang'" >&2
  exit 1
}

require_tool() {
  local tool="$1"
  local purpose="$2"
  if ! command -v "$tool" >/dev/null 2>&1; then
   echo "Error: $tool is required for $purpose generation but is not installed on PATH."
   exit 1
  fi
}

generate_language() {
  local lang
  lang="$(printf '%s' "$1" | tr '[:upper:]' '[:lower:]')"
  local out_dir
  out_dir="$(get_output_dir "$lang")"
  local -a args
  local normalized_flag
  local lang_upper

  lang_upper="$(printf '%s' "$lang" | tr '[:lower:]' '[:upper:]')"
  args=(--proto_path="$PROTO_DIR")

  if [ "$lang" = "go" ]; then
   require_tool protoc-gen-go "Go"
   require_tool protoc-gen-go-grpc "Go"
   args+=(--go_out="$out_dir" --go_opt=module="github.com/iabtechlab/agentic-rtb-framework/pkg")
   args+=(--go_opt="Magenticrtbframework.proto=$GO_ARTF_MODULE")
   args+=(--go_opt="Magenticrtbframeworkservices.proto=$GO_ARTF_MODULE")
   args+=(--go_opt="Mcom/iabtechlab/openrtb/v2/openrtb.proto=$GO_OPENRTB_MODULE")
   args+=(--go-grpc_out="$out_dir" --go-grpc_opt=module="github.com/iabtechlab/agentic-rtb-framework/pkg")
   args+=(--go-grpc_opt="Magenticrtbframework.proto=$GO_ARTF_MODULE")
   args+=(--go-grpc_opt="Magenticrtbframeworkservices.proto=$GO_ARTF_MODULE")
   args+=(--go-grpc_opt="Mcom/iabtechlab/openrtb/v2/openrtb.proto=$GO_OPENRTB_MODULE")
   echo "Generating Go bindings into $out_dir..."
   protoc "${args[@]}" "${PROTO_FILES[@]}"
   echo "Done! Generated Go files in $out_dir"
   return 0
  fi

  if [ "$lang" = "rust" ]; then
   if ! command -v protoc-gen-rust >/dev/null 2>&1 && ! command -v protoc-gen-tonic >/dev/null 2>&1; then
     echo "Error: rust generation requires protoc-gen-rust and/or protoc-gen-tonic in PATH."
     exit 1
   fi
   mkdir -p "$out_dir"
   args+=(--rust_out="$out_dir")
   if command -v protoc-gen-tonic >/dev/null 2>&1; then
     args+=(--tonic_out="$out_dir")
   fi
   echo "Generating Rust bindings into $out_dir..."
   protoc "${args[@]}" "${PROTO_FILES[@]}"
   echo "Done! Generated Rust files in $out_dir"
   return 0
  fi

  mkdir -p "$out_dir"
  echo "Generating ${lang_upper} bindings into $out_dir..."
  protoc "${args[@]}" "--${lang}_out=$out_dir" "${PROTO_FILES[@]}"
  echo "Done! Generated ${lang_upper} files in $out_dir"
}

for lang in "${LANGS[@]}"; do
  normalized="$(printf '%s' "$lang" | tr '[:upper:]' '[:lower:]')"
  if [ "$normalized" = "all" ]; then
   for target in "${SUPPORTED_LANGS[@]}"; do
     generate_language "$target"
   done
  else
   generate_language "$normalized"
  fi
done
