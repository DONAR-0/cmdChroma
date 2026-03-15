#!/usr/bin/env bash
set -euo pipefail

# Build the cmdChroma docker image and optionally extract the built binary
# into the repository's dist/ directory.

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
DIST_DIR="$REPO_ROOT/../dist"

mkdir -p "$DIST_DIR"

# If DOCKER_OUTPUT is set, use buildkit "--output" to export the built binary.
# Otherwise, build a standard docker image.

if [[ -n "${DOCKER_OUTPUT-}" ]]; then
  echo "Building and exporting binary to $DIST_DIR"
  DOCKER_BUILDKIT=1 docker build --output "type=local,dest=$DIST_DIR" -f "$REPO_ROOT/docker/Dockerfile" "$REPO_ROOT"
  echo "✅ Exported binary to $DIST_DIR"
else
  echo "Building docker image 'cmdchroma:latest'"
  docker build -t cmdchroma:latest -f "$REPO_ROOT/docker/Dockerfile" "$REPO_ROOT"
  echo "✅ Image built: cmdchroma:latest"
fi
