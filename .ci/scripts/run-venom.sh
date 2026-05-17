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

# Add the dist folder to the PATH using the absolute path
export PATH="$REPO_ROOT/dist:$PATH"

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

# # Ensure Ollama container is running (commented out - not needed)
# if docker ps -a --filter "name=^/ollama$" --format '{{.Names}}' | grep -q "^ollama$"; then
#     # Container exists, start it if not already running
#     if ! docker ps --filter "name=^/ollama$" --format '{{.Names}}' | grep -q "^ollama$"; then
#         echo "Starting existing Ollama container..."
#         docker start ollama
#     else
#         echo "Ollama container already running."
#     fi
# else
#     echo "Creating and starting Ollama container..."
#     docker run -d --name ollama -p 11434:11434 ollama/ollama:latest
# fi
#
# # Wait for Ollama to be ready
# echo "Waiting for Ollama to be ready on http://localhost:11434..."
# for i in {1..60}; do
#   if curl -s http://localhost:11434/api/tags > /dev/null 2>&1; then
#     echo "Ollama is ready."
#     break
#   fi
#   sleep 1
# done
#
# # Pull the qwen:0.5b model (needed for chat test)
# echo "Ensuring qwen:0.5b model is available..."
# docker exec ollama ollama pull qwen:0.5b || true

echo "Running Venom tests..."
"$VENOM_BIN" run .ci/tests/*.yml \
    --output-dir "$LOG_DIR" \
    -vv
