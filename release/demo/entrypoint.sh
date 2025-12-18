#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

spin_chars=(
  "⠋"
  "⠙"
  "⠹"
  "⠸"
  "⠼"
  "⠴"
  "⠦"
  "⠧"
  "⠇"
  "⠏"
)
spin_index=0
SPINNER_PID=""

start_spinner() {
  local message="$1"
  local spin_index=0

  (
      while true; do
          spin_index=$(( (spin_index + 1) % ${#spin_chars[@]} ))
          # Use printf for reliable carriage return handling
          printf "\r%s %s" "${spin_chars[$spin_index]}" "${message}"
          sleep 0.1
      done
  ) &
  
  SPINNER_PID=$!
  trap 'kill $SPINNER_PID >/dev/null 2>&1' EXIT
}

stop_spinner() {
  kill $SPINNER_PID >/dev/null 2>&1
  # Use printf to clear the line completely (ANSI escape code \033[K)
  printf "\r\033[K"
  trap - EXIT
}

# Defaults for embedded PostgreSQL and RabbitMQ and app
: "${PGDATA:=/var/lib/postgresql/data}"
: "${DB_HOST:=127.0.0.1}"
: "${DB_PORT:=5432}"
: "${DB_NAME:=superplane_demo}"
: "${DB_USERNAME:=postgres}"
: "${DB_PASSWORD:=postgres}"
: "${POSTGRES_DB_SSL:=false}"
: "${APPLICATION_NAME:=superplane}"
: "${BASE_URL:=http://localhost:8000}"
: "${RABBITMQ_URL:=amqp://guest:guest@127.0.0.1:5672}"
: "${SWAGGER_BASE_PATH:=/app/api/swagger}"
: "${RBAC_MODEL_PATH:=/app/rbac/rbac_model.conf}"
: "${RBAC_ORG_POLICY_PATH:=/app/rbac/rbac_org_policy.csv}"
: "${TEMPLATE_DIR:=/app/templates}"
: "${WEB_BASE_PATH:=}"

: "${PUBLIC_API_BASE_PATH:=/api/v1}"

: "${OWNER_SETUP_ENABLED:=yes}"
: "${CLOUDFLARE_QUICK_TUNNEL:=1}"

: "${START_PUBLIC_API:=yes}"
: "${START_INTERNAL_API:=yes}"
: "${START_GRPC_GATEWAY:=yes}"
: "${START_CONSUMERS:=yes}"
: "${START_WEB_SERVER:=yes}"
: "${START_EVENT_DISTRIBUTER:=yes}"
: "${START_WORKFLOW_EVENT_ROUTER:=yes}"
: "${START_WORKFLOW_NODE_EXECUTOR:=yes}"
: "${START_WORKFLOW_NODE_QUEUE_WORKER:=yes}"
: "${START_NODE_REQUEST_WORKER:=yes}"
: "${START_WEBHOOK_PROVISIONER:=yes}"
: "${START_WEBHOOK_CLEANUP_WORKER:=yes}"
: "${START_WORKFLOW_CLEANUP_WORKER:=yes}"

# Reasonable defaults so the server can start without extra config.
: "${ENCRYPTION_KEY:=1234567890abcdefghijklmnopqrstuv}"
: "${JWT_SECRET:=1234567890abcdefghijklmnopqrstuv}"
: "${SESSION_SECRET:=1234567890abcdefghijklmnopqrstuv}"
: "${NO_ENCRYPTION:=yes}"

export DB_HOST DB_PORT DB_NAME DB_USERNAME DB_PASSWORD POSTGRES_DB_SSL APPLICATION_NAME \
  BASE_URL WEB_BASE_PATH PUBLIC_API_BASE_PATH OWNER_SETUP_ENABLED CLOUDFLARE_QUICK_TUNNEL \
  RABBITMQ_URL SWAGGER_BASE_PATH RBAC_MODEL_PATH RBAC_ORG_POLICY_PATH TEMPLATE_DIR \
  START_PUBLIC_API START_INTERNAL_API START_GRPC_GATEWAY START_CONSUMERS \
  START_WEB_SERVER START_EVENT_DISTRIBUTER START_WORKFLOW_EVENT_ROUTER \
  START_WORKFLOW_NODE_EXECUTOR START_WORKFLOW_NODE_QUEUE_WORKER \
  START_NODE_REQUEST_WORKER START_WEBHOOK_PROVISIONER START_WEBHOOK_CLEANUP_WORKER \
  START_WORKFLOW_CLEANUP_WORKER \
  ENCRYPTION_KEY JWT_SECRET SESSION_SECRET NO_ENCRYPTION

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

RABBITMQ_LOG_BASE=${RABBITMQ_LOG_BASE:-/var/log/rabbitmq}
RABBITMQ_MNESIA_BASE=${RABBITMQ_MNESIA_BASE:-/var/lib/rabbitmq/mnesia}

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
  mkdir -p /var/log/cloudflared
  cloudflared tunnel --no-autoupdate --url "http://127.0.0.1:8000" > /var/log/cloudflared/cloudflared.log 2>&1 &

  start_spinner "Creating a public URL via Cloudflare Tunnel"
  for i in {1..30}; do
    if [ -f /var/log/cloudflared/cloudflared.log ]; then
      URL=$(grep -Eo 'https://[a-zA-Z0-9.-]+\.trycloudflare\.com' /var/log/cloudflared/cloudflared.log | head -n 1 || true)
      if [ -n "${URL}" ]; then
        stop_spinner
        break
      fi
    fi
    sleep 2
  done
fi

echo ""
echo "  Visit: ${URL}"

exec /app/build/superplane >/dev/null 2>&1
