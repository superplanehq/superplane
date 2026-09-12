#!/bin/bash

set -euo pipefail
IFS=$'\n\t'

# Run a command in this worktree's Compose `app` container.
# PUBLIC_API_PORT is the UI port. Postgres in this stack is db:5432.

if [[ -f .env ]]; then
  set -a
  # shellcheck disable=SC1091
  . ./.env
  set +a
fi

PORT="${PUBLIC_API_PORT:-8000}"

if [[ ! "$PORT" =~ ^[0-9]+$ ]]; then
  echo "PUBLIC_API_PORT must be a port number. Got: ${PORT}" >&2
  exit 1
fi

if [[ "$#" -lt 1 ]]; then
  echo "Usage: $0 <command> [args...]" >&2
  exit 1
fi

app_id="$(docker compose -f docker-compose.dev.yml ps -q app)"
if [[ -z "$app_id" ]]; then
  echo "The app container is not running in this worktree." >&2
  echo "Run make dev.up first." >&2
  exit 1
fi

project="$(docker inspect -f '{{index .Config.Labels "com.docker.compose.project"}}' "$app_id")"

echo "Using Postgres in Compose project ${project} (db:5432)."
echo "This stack's UI is http://localhost:${PORT}."

exec docker compose -f docker-compose.dev.yml exec \
  -e PUBLIC_API_PORT="$PORT" \
  app \
  "$@"
