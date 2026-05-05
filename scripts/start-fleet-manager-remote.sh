#!/usr/bin/env bash
# Start fleet-manager on a remote Ubuntu host via SSH + Docker (public GHCR image).
#
# Defaults match the fleet-manager EC2 host; override with env vars:
#   SSH_HOST SSH_USER SSH_KEY FLEET_MANAGER_IMAGE HOST_PORT AUTH_TOKEN
#
# Usage:
#   ./scripts/start-fleet-manager-remote.sh
#   AUTH_TOKEN='your-token' ./scripts/start-fleet-manager-remote.sh

set -euo pipefail

SSH_HOST="${SSH_HOST:-13.220.51.216}"
SSH_USER="${SSH_USER:-ubuntu}"
SSH_KEY="${SSH_KEY:-$HOME/.ssh/igor-runners.pem}"
SSH_KEY="${SSH_KEY/#\~/$HOME}"
CONTAINER_NAME="${CONTAINER_NAME:-fleet-manager}"
IMAGE="${FLEET_MANAGER_IMAGE:-ghcr.io/superplanehq/runner/fleet-manager:latest}"
HOST_PORT="${HOST_PORT:-8080}"

if [[ ! -f "$SSH_KEY" ]]; then
  echo "SSH key not found: $SSH_KEY" >&2
  exit 1
fi

echo "Deploying $IMAGE to ${SSH_USER}@${SSH_HOST} (container port 8080 -> host ${HOST_PORT})..."

ssh -i "$SSH_KEY" \
  -o StrictHostKeyChecking=accept-new \
  "${SSH_USER}@${SSH_HOST}" \
  bash -s -- "$IMAGE" "$CONTAINER_NAME" "$HOST_PORT" "${AUTH_TOKEN:-}" << 'REMOTE'
set -euo pipefail
IMAGE="$1"
CONTAINER_NAME="$2"
HOST_PORT="$3"
AUTH_TOKEN="${4:-}"

if ! command -v docker >/dev/null 2>&1; then
  echo "Installing docker.io (needs passwordless sudo on the server)..."
  sudo apt-get update -qq
  sudo DEBIAN_FRONTEND=noninteractive apt-get install -y docker.io
  sudo usermod -aG docker "$(whoami)" 2>/dev/null || true
fi
if docker info >/dev/null 2>&1; then DOCKER=(docker); else DOCKER=(sudo docker); fi

"${DOCKER[@]}" pull "$IMAGE"

"${DOCKER[@]}" stop "$CONTAINER_NAME" 2>/dev/null || true
"${DOCKER[@]}" rm "$CONTAINER_NAME" 2>/dev/null || true

opts=(
  -d
  --name "$CONTAINER_NAME"
  --restart unless-stopped
  -p "${HOST_PORT}:8080"
  -v fleet-manager-data:/home/nonroot
)
if [[ -n "$AUTH_TOKEN" ]]; then
  opts+=( -e "AUTH_TOKEN=${AUTH_TOKEN}" )
fi

"${DOCKER[@]}" run "${opts[@]}" "$IMAGE"

"${DOCKER[@]}" ps --filter "name=${CONTAINER_NAME}"
echo "Health: curl -sS \"http://127.0.0.1:${HOST_PORT}/healthz\""
REMOTE

echo "Done. From your machine: curl -sS \"http://${SSH_HOST}:${HOST_PORT}/healthz\""
