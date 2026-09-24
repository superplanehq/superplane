#!/usr/bin/env bash
# GET /healthz on fleet-manager and task-broker (or override URLs).
#
#   ./scripts/check-deployment-health.sh
#   FLEET_MANAGER_URL=http://… BROKER_URL=http://… ./scripts/check-deployment-health.sh

set -u

FLEET_MANAGER_URL="${FLEET_MANAGER_URL:-http://13.220.51.216:8080}"
BROKER_URL="${BROKER_URL:-http://98.91.210.215:8081}"
TIMEOUT_SEC="${TIMEOUT_SEC:-10}"

die=0

probe() {
  local name="$1" base="$2"
  base="${base%/}"
  printf '%-18s %-40s → ' "$name" "${base}/healthz"
  if curl -fsS --max-time "$TIMEOUT_SEC" "${base}/healthz" >/dev/null; then
    echo OK
  else
    echo FAIL
    die=1
  fi
}

probe "fleet-manager" "$FLEET_MANAGER_URL"
probe "task-broker" "$BROKER_URL"

exit "$die"
