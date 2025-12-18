#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

# Defaults for embedded PostgreSQL and RabbitMQ and app
: "${PGDATA:=/var/lib/postgresql/data}"
: "${DB_HOST:=127.0.0.1}"
: "${DB_PORT:=5432}"
: "${DB_NAME:=superplane_trial}"
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
  BASE_URL WEB_BASE_PATH PUBLIC_API_BASE_PATH \
  RABBITMQ_URL SWAGGER_BASE_PATH RBAC_MODEL_PATH RBAC_ORG_POLICY_PATH TEMPLATE_DIR \
  START_PUBLIC_API START_INTERNAL_API START_GRPC_GATEWAY START_CONSUMERS \
  START_WEB_SERVER START_EVENT_DISTRIBUTER START_WORKFLOW_EVENT_ROUTER \
  START_WORKFLOW_NODE_EXECUTOR START_WORKFLOW_NODE_QUEUE_WORKER \
  START_NODE_REQUEST_WORKER START_WEBHOOK_PROVISIONER START_WEBHOOK_CLEANUP_WORKER \
  START_WORKFLOW_CLEANUP_WORKER \
  ENCRYPTION_KEY JWT_SECRET SESSION_SECRET NO_ENCRYPTION

echo "Starting embedded PostgreSQL for trial..."

# Initialize cluster if needed
if [ ! -s "${PGDATA}/PG_VERSION" ]; then
  echo "Initializing PostgreSQL data directory at ${PGDATA}..."
  mkdir -p "${PGDATA}"
  chown -R postgres:postgres "${PGDATA}" || true

  su - postgres -c "initdb -D '${PGDATA}' -E UTF8 --auth=trust"
fi

# Start postgres in the background
su - postgres -c "pg_ctl -D '${PGDATA}' -o '-p ${DB_PORT}' -w start"

echo "PostgreSQL started on ${DB_HOST}:${DB_PORT}"

echo "Database server ready on ${DB_HOST}:${DB_PORT}."

echo "Starting embedded RabbitMQ for trial..."

RABBITMQ_LOG_BASE=${RABBITMQ_LOG_BASE:-/var/log/rabbitmq}
RABBITMQ_MNESIA_BASE=${RABBITMQ_MNESIA_BASE:-/var/lib/rabbitmq/mnesia}

mkdir -p "${RABBITMQ_LOG_BASE}" "${RABBITMQ_MNESIA_BASE}"

rabbitmq-server -detached

echo "Waiting for RabbitMQ to be ready..."
for i in {1..30}; do
  if rabbitmq-diagnostics -q ping >/dev/null 2>&1; then
    echo "RabbitMQ is up."
    break
  fi
  sleep 1
done

echo "Running Superplane migrations and starting server..."

exec /app/docker-entrypoint.sh
