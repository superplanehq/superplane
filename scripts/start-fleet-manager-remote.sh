#!/usr/bin/env bash
# Start fleet-manager on a remote Ubuntu host via SSH + Docker (GHCR images from CI).
#
# Required on the SSH side: outbound HTTPS (docker pull); for EC2 hot pool, IAM instance
# profile on the host works without embedding AWS_ACCESS_KEY_* in env.
#
# Local env (override defaults):
#   SSH_HOST SSH_USER SSH_KEY CONTAINER_NAME HOST_PORT
#   FLEET_MANAGER_IMAGE         — default ghcr.io/superplanehq/runner/fleet-manager:latest
#                                 Pin a digest or tag after builds, e.g. :main-<sha>
#   FLEET_MANAGER_CONFIG_FILE   — REQUIRED. Local path to the JSON config (uploaded and
#                                 bind-mounted into the container at /etc/fleet-manager/config.json).
#                                 Template: scripts/deploy/fleet-manager.config.example.json.
#
# The config file carries everything the FM needs (broker URL, AWS region, auth tokens,
# diagnostics token, listen addr, per-pool AMIs/instance types/headroom). No --env-file,
# no -e AUTH_TOKEN passthrough.
#
# Usage:
#   cp scripts/deploy/fleet-manager.config.example.json scripts/deploy/fleet-manager.config.json
#   $EDITOR scripts/deploy/fleet-manager.config.json
#   FLEET_MANAGER_CONFIG_FILE=scripts/deploy/fleet-manager.config.json ./scripts/start-fleet-manager-remote.sh
#   FLEET_MANAGER_IMAGE='ghcr.io/superplanehq/runner/fleet-manager:abc123' FLEET_MANAGER_CONFIG_FILE=… ./scripts/start-fleet-manager-remote.sh

set -euo pipefail

SSH_HOST="${SSH_HOST:-13.220.51.216}"
SSH_USER="${SSH_USER:-ubuntu}"
SSH_KEY="${SSH_KEY:-$HOME/.ssh/igor-runners.pem}"
SSH_KEY="${SSH_KEY/#\~/$HOME}"
CONTAINER_NAME="${CONTAINER_NAME:-fleet-manager}"
IMAGE="${FLEET_MANAGER_IMAGE:-ghcr.io/superplanehq/runner/fleet-manager:latest}"
HOST_PORT="${HOST_PORT:-8080}"

REMOTE_CONFIG_PATH="${REMOTE_CONFIG_PATH:-/tmp/fleet-manager.config.json}"
CONTAINER_CONFIG_PATH="${CONTAINER_CONFIG_PATH:-/etc/fleet-manager/config.json}"

if [[ ! -f "$SSH_KEY" ]]; then
  echo "SSH key not found: $SSH_KEY" >&2
  exit 1
fi

if [[ -z "${FLEET_MANAGER_CONFIG_FILE:-}" ]]; then
  echo "FLEET_MANAGER_CONFIG_FILE is required (path to the JSON config to upload)" >&2
  echo "Template: scripts/deploy/fleet-manager.config.example.json" >&2
  exit 1
fi
if [[ ! -f "$FLEET_MANAGER_CONFIG_FILE" ]]; then
  echo "FLEET_MANAGER_CONFIG_FILE not found: $FLEET_MANAGER_CONFIG_FILE" >&2
  exit 1
fi

echo "Uploading config → ${SSH_USER}@${SSH_HOST}:${REMOTE_CONFIG_PATH}"
scp -i "$SSH_KEY" -o StrictHostKeyChecking=accept-new \
  "$FLEET_MANAGER_CONFIG_FILE" "${SSH_USER}@${SSH_HOST}:${REMOTE_CONFIG_PATH}"

echo "Deploying $IMAGE to ${SSH_USER}@${SSH_HOST} (container port 8080 -> host ${HOST_PORT})..."

ssh -i "$SSH_KEY" \
  -o StrictHostKeyChecking=accept-new \
  "${SSH_USER}@${SSH_HOST}" \
  bash -s -- "$IMAGE" "$CONTAINER_NAME" "$HOST_PORT" "$REMOTE_CONFIG_PATH" "$CONTAINER_CONFIG_PATH" << 'REMOTE'
set -euo pipefail
IMAGE="$1"
CONTAINER_NAME="$2"
HOST_PORT="$3"
REMOTE_CONFIG_PATH="$4"
CONTAINER_CONFIG_PATH="$5"

if ! command -v docker >/dev/null 2>&1; then
  echo "Installing docker.io (needs passwordless sudo on the server)..."
  sudo apt-get update -qq
  sudo DEBIAN_FRONTEND=noninteractive apt-get install -y docker.io
  sudo usermod -aG docker "$(whoami)" 2>/dev/null || true
fi
if docker info >/dev/null 2>&1; then DOCKER=(docker); else DOCKER=(sudo docker); fi

# Config file holds the runner_auth_token etc. — keep it tight.
chmod 600 "$REMOTE_CONFIG_PATH" 2>/dev/null || true

"${DOCKER[@]}" pull "$IMAGE"

"${DOCKER[@]}" stop "$CONTAINER_NAME" 2>/dev/null || true
"${DOCKER[@]}" rm "$CONTAINER_NAME" 2>/dev/null || true

"${DOCKER[@]}" run \
  -d \
  --name "$CONTAINER_NAME" \
  --restart unless-stopped \
  -p "${HOST_PORT}:8080" \
  -v "${REMOTE_CONFIG_PATH}:${CONTAINER_CONFIG_PATH}:ro" \
  "$IMAGE"

"${DOCKER[@]}" ps --filter "name=${CONTAINER_NAME}"
echo "Health: curl -sS \"http://127.0.0.1:${HOST_PORT}/healthz\""
REMOTE

echo "Done. From your machine: curl -sS \"http://${SSH_HOST}:${HOST_PORT}/healthz\""
