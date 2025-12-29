#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

# Source spinner functions
source /app/spinner.sh

# ===========================================================================
# Setting up data directory structure
# ===========================================================================

# Ensure /app/data exists and has proper permissions for persistent storage
mkdir -p /app/data/postgres /app/data/rabbitmq/mnesia /app/data/rabbitmq/logs /app/data/cloudflared
chown -R postgres:postgres /app/data/postgres || true

# ===========================================================================
# Generating and loading environment variables
# ===========================================================================

# Generate or load all environment variables
/app/gen-superplane-env.sh /app/data/superplane.env

# Source the environment file
set -a
source /app/data/superplane.env
set +a

# ===========================================================================
# Starting embedded services
# ===========================================================================

start_spinner "Starting services"

if [ ! -s "${PGDATA}/PG_VERSION" ]; then
  mkdir -p "${PGDATA}"
  chown -R postgres:postgres "${PGDATA}" || true

  su - postgres -c "initdb -D '${PGDATA}' -E UTF8 --auth=trust" >/dev/null 2>&1
fi

su - postgres -c "pg_ctl -D '${PGDATA}' -o '-p ${DB_PORT}' -w start" >/dev/null 2>&1

RABBITMQ_LOG_BASE=${RABBITMQ_LOG_BASE:-/app/data/rabbitmq/logs}
RABBITMQ_MNESIA_BASE=${RABBITMQ_MNESIA_BASE:-/app/data/rabbitmq/mnesia}

mkdir -p "${RABBITMQ_LOG_BASE}" "${RABBITMQ_MNESIA_BASE}"

rabbitmq-server -detached
for i in {1..30}; do
  if rabbitmq-diagnostics -q ping >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

stop_spinner

# ===========================================================================
# Running DB migrations
# ===========================================================================

start_spinner "Running database migrations"

if [ -z "${DB_PASSWORD:-}" ]; then echo "DB username not set" && exit 1; fi
if [ -z "${DB_HOST:-}" ]; then echo "DB host not set" && exit 1; fi
if [ -z "${DB_PORT:-}" ]; then echo "DB port not set" && exit 1; fi
if [ -z "${DB_USERNAME:-}" ]; then echo "DB username not set" && exit 1; fi
if [ -z "${DB_NAME:-}" ]; then echo "DB name not set" && exit 1; fi
if [ -z "${APPLICATION_NAME:-}" ]; then echo "Application name not set" && exit 1; fi

if [ "${POSTGRES_DB_SSL}" = "true" ]; then
  export PGSSLMODE=require
else
  export PGSSLMODE=disable
fi

PGPASSWORD="${DB_PASSWORD}" createdb -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USERNAME}" "${DB_NAME}" >/dev/null 2>&1 || true

DB_URL="postgres://${DB_USERNAME}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${PGSSLMODE}"
migrate -source file:///app/db/migrations -database "${DB_URL}" up >/dev/null 2>&1

stop_spinner

# ===========================================================================
# Starting Cloudflare tunnel
# ===========================================================================

if [ "${CLOUDFLARE_QUICK_TUNNEL}" = "1" ]; then
  mkdir -p /app/data/cloudflared
  cloudflared tunnel --no-autoupdate --url "http://127.0.0.1:8000" > /app/data/cloudflared/cloudflared.log 2>&1 &

  start_spinner "Creating a public URL via Cloudflare Tunnel"
  for i in {1..60}; do
    if [ -f /app/data/cloudflared/cloudflared.log ]; then
      URL=$(grep -Eo 'https://[a-zA-Z0-9.-]+\.trycloudflare\.com' /app/data/cloudflared/cloudflared.log | head -n 1 || true)
      if [ -n "${URL}" ]; then
        BASE_URL="${URL}"
        export BASE_URL
        stop_spinner
        break
      fi
    fi
    sleep 1
  done
fi

echo ""
echo "  Visit: ${URL}"

exec /app/build/superplane >/dev/null 2>&1