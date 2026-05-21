#!/usr/bin/env bash
# Start fleet-manager on a remote Ubuntu host via SSH + Docker (GHCR images from CI).
#
# Required on the SSH side: outbound HTTPS (docker pull); for EC2 hot pool, IAM instance
# profile on the host works without embedding AWS_ACCESS_KEY_* in env.
#
# Local env (override defaults):
#   SSH_HOST SSH_USER SSH_KEY CONTAINER_NAME HOST_PORT
#   FLEET_MANAGER_IMAGE   — default ghcr.io/superplanehq/runner/fleet-manager:latest
#                          Pin a digest or tag after builds, e.g. :main-<sha>
#   AUTH_TOKEN            — passed into the container (-e AUTH_TOKEN)
#
# Optional extra container env (recommended for AWS/EC2): set FLEET_MANAGER_ENV_FILE to a
# local path; it is uploaded and passed as docker --env-file. Typical keys:
#   AWS_REGION | AWS_DEFAULT_REGION
#   EC2_PROVISION_HOT_INSTANCE_COUNT    — unset this file entirely to disable EC2 pool
#   EC2_PROVISION_AMI_ID
#   EC2_PROVISION_SUBNET_ID
#   EC2_PROVISION_SECURITY_GROUP_IDS    — comma-separated
#   EC2_PROVISION_FLEET_MANAGER_URL       — URL runner instances use (often private VPC)
#   EC2_PROVISION_INSTANCE_TYPE EC2_PROVISION_RUNNER_S3_URI EC2_PROVISION_RUNNER_AUTH_TOKEN EC2_PROVISION_RUNNER_INSTANCE_PROFILE
#   EC2_PROVISION_KEY_NAME EC2_PROVISION_RUNNER_INSTANCE_PROFILE
#   EC2_PROVISION_RECONCILE_INTERVAL_SEC   REAP_INTERVAL_SEC
#
# Ready-made env templates: scripts/deploy/fleet-manager.env (gitignored) and fleet-manager.env.example (checked in).
#
# Usage:
#   cp scripts/deploy/fleet-manager.env.example scripts/deploy/fleet-manager.env && $EDITOR scripts/deploy/fleet-manager.env
#   AUTH_TOKEN='…' FLEET_MANAGER_ENV_FILE=scripts/deploy/fleet-manager.env ./scripts/start-fleet-manager-remote.sh
#   FLEET_MANAGER_IMAGE='ghcr.io/superplanehq/runner/fleet-manager:abc123' ./scripts/start-fleet-manager-remote.sh

set -euo pipefail

SSH_HOST="${SSH_HOST:-13.220.51.216}"
SSH_USER="${SSH_USER:-ubuntu}"
SSH_KEY="${SSH_KEY:-$HOME/.ssh/igor-runners.pem}"
SSH_KEY="${SSH_KEY/#\~/$HOME}"
CONTAINER_NAME="${CONTAINER_NAME:-fleet-manager}"
IMAGE="${FLEET_MANAGER_IMAGE:-ghcr.io/superplanehq/runner/fleet-manager:latest}"
HOST_PORT="${HOST_PORT:-8080}"

REMOTE_ENV_PATH="${REMOTE_ENV_PATH:-/tmp/fleet-manager.deploy.env}"
USE_ENVFILE=0

if [[ ! -f "$SSH_KEY" ]]; then
  echo "SSH key not found: $SSH_KEY" >&2
  exit 1
fi

if [[ -n "${FLEET_MANAGER_ENV_FILE:-}" ]]; then
  if [[ ! -f "$FLEET_MANAGER_ENV_FILE" ]]; then
    echo "FLEET_MANAGER_ENV_FILE not found: $FLEET_MANAGER_ENV_FILE" >&2
    exit 1
  fi
  echo "Uploading env file → ${SSH_USER}@${SSH_HOST}:${REMOTE_ENV_PATH}"
  scp -i "$SSH_KEY" -o StrictHostKeyChecking=accept-new \
    "$FLEET_MANAGER_ENV_FILE" "${SSH_USER}@${SSH_HOST}:${REMOTE_ENV_PATH}"
  USE_ENVFILE=1
fi

echo "Deploying $IMAGE to ${SSH_USER}@${SSH_HOST} (container port 8080 -> host ${HOST_PORT})..."

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
  -p "${HOST_PORT}:8080"
  -v fleet-manager-data:/home/nonroot
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
