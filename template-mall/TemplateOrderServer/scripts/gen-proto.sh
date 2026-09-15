#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/api/gen/templateorder/v1"
mkdir -p "$OUT"
protoc \
  --proto_path="$ROOT/api/proto" \
  --go_out="$OUT" --go_opt=paths=source_relative \
  --go-grpc_out="$OUT" --go-grpc_opt=paths=source_relative \
  "$ROOT"/api/proto/*.proto
echo "proto generated."
