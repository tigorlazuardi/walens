#!/bin/bash
# Generate TypeScript schema types from the OpenAPI specification.
#
# Usage:
#   ./scripts/generate-api-schema.sh
#
# Runs `walens openapi-yaml` and pipes the output directly to openapi-typescript.

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FRONTEND_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
ROOT_DIR="$(cd "$FRONTEND_DIR/.." && pwd)"
OUTPUT_PATH="$FRONTEND_DIR/src/lib/api/generated/schema.ts"

echo "Generating API schema..."

# Ensure the output directory exists
mkdir -p "$(dirname "$OUTPUT_PATH")"

# Generate TypeScript schema from the WALENS backend OpenAPI YAML output via a FIFO pipe.
# This avoids a persistent temp file while still supporting openapi-typescript's
# file-based input requirement.
cd "$ROOT_DIR"
FIFO="$(mkfifo "$ROOT_DIR/openapi-yaml-pipe" && echo "$ROOT_DIR/openapi-yaml-pipe")"
go run ./cmd/walens openapi-yaml > "$FIFO" &
npx openapi-typescript "$FIFO" -o "$OUTPUT_PATH"
rm -f "$FIFO"

echo "Schema generated successfully: $OUTPUT_PATH"
