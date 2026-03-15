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
# Path to your config file
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

echo "Running Venom tests..."
"$VENOM_BIN" run .ci/tests/*.yml \
    --output-dir "$LOG_DIR" \
    -vv
