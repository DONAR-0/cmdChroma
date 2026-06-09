#!/usr/bin/env bash

# Exit on error + unset vars + pipefail
set -euo pipefail
IFS=$'\n\t'

# Base paths
REPO_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
BIN_DIR="$REPO_ROOT/.ci/bin"
LOG_DIR="$REPO_ROOT/.ci/logs"

# Configuration
VENOM_VERSION="v1.1.0"
OS="linux"
ARCH="amd64"
BINARY_URL="https://github.com/ovh/venom/releases/download/${VENOM_VERSION}/venom.${OS}-${ARCH}"
VENOM_CONFIG="$REPO_ROOT/.venomrc.yml"
VENOM_BIN="${BIN_DIR}/venom"

# Clean logs directory
rm -rf "$LOG_DIR"

# Create Directories
mkdir -p "$BIN_DIR"
mkdir -p "$LOG_DIR"

# Download Venom if missing
if [ ! -f "$VENOM_BIN" ]; then
  echo "Downloading Venom..."
  curl -L "$BINARY_URL" -o "$VENOM_BIN"
  chmod +x "$VENOM_BIN"
fi

# Add the dist and .ci/bin folders to the PATH using the absolute path
export PATH="$REPO_ROOT/dist:$REPO_ROOT/.ci/bin:$PATH"

# Also keep this for safety until RPATH is 100% verified
export LD_LIBRARY_PATH="$REPO_ROOT/models/onnx_runtime/lib:${LD_LIBRARY_PATH:-}"

# Ensure ChromaDB is running
echo "Checking for ChromaDB..."
if ! curl -s http://localhost:8000/api/v2/heartbeat > /dev/null 2>&1; then
  echo "Starting ChromaDB via Docker Compose..."
  docker compose -f "$REPO_ROOT/.ci/chroma.compose.yaml" up -d chromadb
  # Wait for ChromaDB to be ready
  echo "Waiting for ChromaDB to be ready..."
  for i in {1..30}; do
    if curl -s http://localhost:8000/api/v2/heartbeat > /dev/null 2>&1; then
      echo "ChromaDB is ready."
      break
    fi
    sleep 1
  done
else
  echo "ChromaDB is already running."
fi

# Run Venom tests with TEST_FILES support
echo "Running Venom tests..."
if [ -n "${TEST_FILES:-}" ]; then
  echo "Running specific test files: $TEST_FILES"
  "$VENOM_BIN" run ${TEST_FILES} \
  --output-dir "$LOG_DIR" \
  -vv
else
  "$VENOM_BIN" run .ci/tests/*.yml \
  --output-dir "$LOG_DIR" \
  -vv
fi
