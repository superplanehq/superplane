#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

if [ -z "${DB_PASSWORD:-}" ]; then echo "DB_PASSWORD is not set" && exit 1; fi
if [ -z "${DB_HOST:-}" ]; then echo "DB_HOST is not set" && exit 1; fi
if [ -z "${DB_PORT:-}" ]; then echo "DB_PORT is not set" && exit 1; fi
if [ -z "${DB_USERNAME:-}" ]; then echo "DB_USERNAME is not set" && exit 1; fi
if [ -z "${DB_NAME:-}" ]; then echo "DB_NAME is not set" && exit 1; fi

if [ -z "${APPLICATION_NAME:-}" ]; then
  APPLICATION_NAME=agent
fi
export APPLICATION_NAME

# Agent runtime uses DB_SSLMODE; keep POSTGRES_DB_SSL for parity with the app entrypoint.
if [ -n "${DB_SSLMODE:-}" ]; then
  PGSSLMODE="${DB_SSLMODE}"
elif [ "${POSTGRES_DB_SSL:-}" = "true" ]; then
  PGSSLMODE=require
else
  PGSSLMODE=disable
fi

echo "Creating DB..."
PGPASSWORD="${DB_PASSWORD}" createdb -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USERNAME}" "${DB_NAME}" || true

echo "Migrating DB..."
DB_URL="postgres://${DB_USERNAME}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${PGSSLMODE}"
migrate -source file:///app/agent/db/migrations -database "${DB_URL}" up

APP_PORT="${APP_PORT:-8090}"
UVICORN_GRACEFUL_SHUTDOWN_TIMEOUT="${UVICORN_GRACEFUL_SHUTDOWN_TIMEOUT:-310}"

echo "Starting agent server..."
exec uvicorn ai.web:create_app \
  --factory \
  --host 0.0.0.0 \
  --port "${APP_PORT}" \
  --timeout-graceful-shutdown "${UVICORN_GRACEFUL_SHUTDOWN_TIMEOUT}"
