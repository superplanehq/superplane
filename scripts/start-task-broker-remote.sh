#!/usr/bin/env bash
# Start task-broker on a remote Ubuntu host via SSH + Docker (public GHCR image).
#
# Defaults match igor-runner EC2; override with env vars:
#   SSH_HOST SSH_USER SSH_KEY TASK_BROKER_IMAGE HOST_PORT AUTH_TOKEN BROKER_PUBLIC_URL
#
# BROKER_PUBLIC_URL — base URL fleet-managers use to reach this broker (completion webhooks).
# If unset, defaults to http://$SSH_HOST:$HOST_PORT (fine for first wiring over plain HTTP).
#
# Usage:
#   ./scripts/start-task-broker-remote.sh
#   AUTH_TOKEN='your-token' BROKER_PUBLIC_URL='https://broker.example.com' ./scripts/start-task-broker-remote.sh

set -euo pipefail

SSH_HOST="${SSH_HOST:-98.91.210.215}"
SSH_USER="${SSH_USER:-ubuntu}"
SSH_KEY="${SSH_KEY:-$HOME/.ssh/igor-runners.pem}"
SSH_KEY="${SSH_KEY/#\~/$HOME}"
CONTAINER_NAME="${CONTAINER_NAME:-task-broker}"
IMAGE="${TASK_BROKER_IMAGE:-ghcr.io/superplanehq/runner/task-broker:latest}"
HOST_PORT="${HOST_PORT:-8081}"
BROKER_PUBLIC_URL="${BROKER_PUBLIC_URL:-http://${SSH_HOST}:${HOST_PORT}}"

if [[ ! -f "$SSH_KEY" ]]; then
  echo "SSH key not found: $SSH_KEY" >&2
  exit 1
fi

echo "Deploying $IMAGE to ${SSH_USER}@${SSH_HOST} (container port 8081 -> host ${HOST_PORT})..."
echo "BROKER_PUBLIC_URL=$BROKER_PUBLIC_URL"

ssh -i "$SSH_KEY" \
  -o StrictHostKeyChecking=accept-new \
  "${SSH_USER}@${SSH_HOST}" \
  bash -s -- "$IMAGE" "$CONTAINER_NAME" "$HOST_PORT" "${AUTH_TOKEN:-}" "$BROKER_PUBLIC_URL" << 'REMOTE'
set -euo pipefail
IMAGE="$1"
CONTAINER_NAME="$2"
HOST_PORT="$3"
AUTH_TOKEN="${4:-}"
BROKER_PUBLIC_URL="${5:-}"

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
  -p "${HOST_PORT}:8081"
  -e "BROKER_PUBLIC_URL=${BROKER_PUBLIC_URL}"
  -v task-broker-data:/home/nonroot
)
if [[ -n "$AUTH_TOKEN" ]]; then
  opts+=( -e "AUTH_TOKEN=${AUTH_TOKEN}" )
fi

"${DOCKER[@]}" run "${opts[@]}" "$IMAGE"

"${DOCKER[@]}" ps --filter "name=${CONTAINER_NAME}"
echo "Health: curl -sS \"http://127.0.0.1:${HOST_PORT}/healthz\""
REMOTE

echo "Done. From your machine: curl -sS \"http://${SSH_HOST}:${HOST_PORT}/healthz\""
