#!/usr/bin/env bash
# Start task-broker on a remote Ubuntu host via SSH + Docker (GHCR images from CI).
#
# Local env (override defaults):
#   SSH_HOST SSH_USER SSH_KEY CONTAINER_NAME HOST_PORT
#   TASK_BROKER_IMAGE — default ghcr.io/superplanehq/runner/task-broker:latest
#
#   AUTH_TOKEN       — bearer token for broker /v1 (required; or set in TASK_BROKER_ENV_FILE)
#
# Optional extra env for the container: TASK_BROKER_ENV_FILE → uploaded; docker --env-file
# (LISTEN_ADDR, DATABASE_URL, etc.—DATABASE_URL is required in the env file or image env).
#
# Usage:
#   TASK_BROKER_ENV_FILE=scripts/deploy/task-broker.env ./scripts/start-task-broker-remote.sh

set -euo pipefail

SSH_HOST="${SSH_HOST:-100.24.107.156}"
SSH_USER="${SSH_USER:-ubuntu}"
SSH_KEY="${SSH_KEY:-$HOME/.ssh/igor-runners.pem}"
SSH_KEY="${SSH_KEY/#\~/$HOME}"
CONTAINER_NAME="${CONTAINER_NAME:-task-broker}"
IMAGE="${TASK_BROKER_IMAGE:-ghcr.io/superplanehq/runner/task-broker:latest}"
HOST_PORT="${HOST_PORT:-8081}"

REMOTE_ENV_PATH="${REMOTE_ENV_PATH:-/tmp/task-broker.deploy.env}"
USE_ENVFILE=0

if [[ ! -f "$SSH_KEY" ]]; then
  echo "SSH key not found: $SSH_KEY" >&2
  exit 1
fi

if [[ -n "${TASK_BROKER_ENV_FILE:-}" ]]; then
  if [[ ! -f "$TASK_BROKER_ENV_FILE" ]]; then
    echo "TASK_BROKER_ENV_FILE not found: $TASK_BROKER_ENV_FILE" >&2
    exit 1
  fi
  echo "Uploading env file → ${SSH_USER}@${SSH_HOST}:${REMOTE_ENV_PATH}"
  scp -i "$SSH_KEY" -o StrictHostKeyChecking=accept-new \
    "$TASK_BROKER_ENV_FILE" "${SSH_USER}@${SSH_HOST}:${REMOTE_ENV_PATH}"
  USE_ENVFILE=1
fi

echo "Deploying $IMAGE to ${SSH_USER}@${SSH_HOST} (container port 8081 -> host ${HOST_PORT})..."

ssh -i "$SSH_KEY" \
  -o StrictHostKeyChecking=accept-new \
  "${SSH_USER}@${SSH_HOST}" \
  bash -s -- "$IMAGE" "$CONTAINER_NAME" "$HOST_PORT" "$USE_ENVFILE" "$REMOTE_ENV_PATH" "${AUTH_TOKEN:-}" << 'REMOTE'
set -euo pipefail
IMAGE="$1"
CONTAINER_NAME="$2"
HOST_PORT="$3"
USE_ENVFILE="${4:-0}"
REMOTE_ENV_PATH="${5:-}"
AUTH_TOKEN="${6:-}"

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
)
if [[ -n "$AUTH_TOKEN" ]]; then
  opts+=( -e "AUTH_TOKEN=${AUTH_TOKEN}" )
fi
if [[ "$USE_ENVFILE" == "1" ]]; then
  chmod 600 "$REMOTE_ENV_PATH" 2>/dev/null || chmod 0644 "$REMOTE_ENV_PATH"
  opts+=( --env-file "$REMOTE_ENV_PATH" )
fi

"${DOCKER[@]}" run "${opts[@]}" "$IMAGE"

"${DOCKER[@]}" ps --filter "name=${CONTAINER_NAME}"
echo "Health: curl -sS \"http://127.0.0.1:${HOST_PORT}/healthz\""
REMOTE

echo "Done. From your machine: curl -sS \"http://${SSH_HOST}:${HOST_PORT}/healthz\""
