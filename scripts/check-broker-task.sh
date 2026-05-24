#!/usr/bin/env bash
# Inspect task state via task-broker GET /v1/tasks/{id}.
#
# Uses AUTH_TOKEN and optional BROKER_URL from scripts/deploy/task-broker.env when
# TASK_BROKER_ENV_FILE points at that file (default). Override with AUTH_TOKEN / BROKER_URL.
#
# Usage:
#   ./scripts/check-broker-task.sh <broker_task_id>
#   TASK_BROKER_ENV_FILE=/path/task-broker.env ./scripts/check-broker-task.sh <broker_task_id>
#   AUTH_TOKEN='…' BROKER_URL='http://broker:8081' ./scripts/check-broker-task.sh <broker_task_id>

set -euo pipefail

TASK_ID="${1:-}"
if [[ -z "$(echo "${TASK_ID}" | tr -d '[:space:]')" ]]; then
  echo "usage: $(basename "$0") <broker_task_id>" >&2
  exit 2
fi

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ENV_FILE="${TASK_BROKER_ENV_FILE:-$ROOT/scripts/deploy/task-broker.env}"

load_kv() {
  local key="$1" file="$2"
  grep -E "^${key}=" "$file" 2>/dev/null | head -1 | cut -d= -f2- | tr -d '\r' || true
}

if [[ -f "$ENV_FILE" ]]; then
  if [[ -z "${BROKER_URL:-}" ]]; then
    export BROKER_URL
    BROKER_URL="$(load_kv BROKER_URL "$ENV_FILE")"
    if [[ "$(echo "$BROKER_URL" | tr -d '[:space:]')" == "" ]]; then
      unset BROKER_URL
    fi
  fi
  if [[ -z "${AUTH_TOKEN:-}" ]]; then
    export AUTH_TOKEN
    AUTH_TOKEN="$(load_kv AUTH_TOKEN "$ENV_FILE")"
  fi
fi

BROKER_URL="${BROKER_URL:-http://98.91.210.215:8081}"
BROKER_URL="${BROKER_URL%/}"

if [[ "$(echo "${AUTH_TOKEN:-}" | tr -d '[:space:]')" == "" ]]; then
  echo "AUTH_TOKEN missing: set env AUTH_TOKEN or add it to TASK_BROKER_ENV_FILE ($ENV_FILE)" >&2
  exit 1
fi

tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT
CODE="$(
  curl -sS -o "$tmp" -w '%{http_code}' --max-time 15 \
    -H "Authorization: Bearer ${AUTH_TOKEN}" \
    "${BROKER_URL}/v1/tasks/$(echo "${TASK_ID}" | tr -d '[:space:]')"
)"
BODY="$(cat "$tmp")"
echo "HTTP ${CODE}"
if command -v jq >/dev/null 2>&1; then
  echo "$BODY" | jq . 2>/dev/null || echo "$BODY"
else
  echo "$BODY"
fi
if [[ "$CODE" != 200 ]]; then
  exit 1
fi
