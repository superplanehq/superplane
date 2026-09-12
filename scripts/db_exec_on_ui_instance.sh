#!/bin/bash

set -euo pipefail
IFS=$'\n\t'

# Run a command in the Compose `app` container that publishes PUBLIC_API_PORT.
# That port is the UI, not Postgres. Inside the container, Postgres is db:5432.

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

ids="$(docker ps -q \
  --filter "label=com.docker.compose.service=app" \
  --filter "publish=${PORT}")"

if [[ -z "$ids" ]]; then
  echo "No SuperPlane app container is publishing port ${PORT}." >&2
  echo "Set PUBLIC_API_PORT in .env to the UI port of the running instance." >&2
  echo "Then run make dev.up in that worktree." >&2
  exit 1
fi

# Host bash 3.2: split on IFS, not mapfile.
# shellcheck disable=SC2086
set -- $ids
if [[ "$#" -gt 1 ]]; then
  echo "More than one app container publishes port ${PORT}." >&2
  exit 1
fi

cid="$1"
project="$(docker inspect -f '{{index .Config.Labels "com.docker.compose.project"}}' "$cid")"

echo "Using the database for http://localhost:${PORT} (Compose project ${project})."

exec docker exec -e PUBLIC_API_PORT="$PORT" "$cid" "$@"
