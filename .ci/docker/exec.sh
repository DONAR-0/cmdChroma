#!/usr/bin/env bash
set -euo pipefail

# Helper to run cmdChroma inside a running container.
# If the container is not running, it will start one in the background.

CONTAINER_NAME="${CONTAINER_NAME:-cmdchroma}"
IMAGE_NAME="${IMAGE_NAME:-cmdchroma:latest}"

# Ensure the image exists
if ! docker image inspect "$IMAGE_NAME" > /dev/null 2>&1; then
  echo "Docker image '$IMAGE_NAME' not found. Build it first with:"
  echo "  ./.ci/docker/build.sh"
  exit 1
fi

# Start container if it's not already running
if ! docker ps --filter "name=^/${CONTAINER_NAME}$" --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
  echo "Starting container '$CONTAINER_NAME' (detached)..."
  docker run -d --rm --name "$CONTAINER_NAME" "$IMAGE_NAME" sleep infinity
fi

# Run the command inside the container
docker exec -it "$CONTAINER_NAME" cmdChroma "$@"