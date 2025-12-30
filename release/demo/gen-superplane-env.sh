#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

# Default env file path
ENV_FILE="${1:-/app/data/superplane.env}"

# Function to generate a random 32-character hex string
generate_secret() {
  head -c 16 /dev/urandom | od -An -tx1 | tr -d ' \n'
}

# Set default values for all environment variables
PGDATA="${PGDATA:-/app/data/postgres}"
DB_HOST="${DB_HOST:-127.0.0.1}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-superplane_demo}"
DB_USERNAME="${DB_USERNAME:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-postgres}"
POSTGRES_DB_SSL="${POSTGRES_DB_SSL:-false}"
APPLICATION_NAME="${APPLICATION_NAME:-superplane}"
BASE_URL="${BASE_URL:-http://localhost:3000}"
WEBHOOKS_BASE_URL="${WEBHOOKS_BASE_URL:-}"
RABBITMQ_URL="${RABBITMQ_URL:-amqp://guest:guest@127.0.0.1:5672}"
SWAGGER_BASE_PATH="${SWAGGER_BASE_PATH:-/app/api/swagger}"
RBAC_MODEL_PATH="${RBAC_MODEL_PATH:-/app/rbac/rbac_model.conf}"
RBAC_ORG_POLICY_PATH="${RBAC_ORG_POLICY_PATH:-/app/rbac/rbac_org_policy.csv}"
TEMPLATE_DIR="${TEMPLATE_DIR:-/app/templates}"
WEB_BASE_PATH="${WEB_BASE_PATH:-}"
PUBLIC_API_BASE_PATH="${PUBLIC_API_BASE_PATH:-/api/v1}"
OWNER_SETUP_ENABLED="${OWNER_SETUP_ENABLED:-yes}"
LOCALTUNNEL_ENABLED="${LOCALTUNNEL_ENABLED:-1}"
START_PUBLIC_API="${START_PUBLIC_API:-yes}"
START_INTERNAL_API="${START_INTERNAL_API:-yes}"
START_GRPC_GATEWAY="${START_GRPC_GATEWAY:-yes}"
START_CONSUMERS="${START_CONSUMERS:-yes}"
START_WEB_SERVER="${START_WEB_SERVER:-yes}"
START_EVENT_DISTRIBUTER="${START_EVENT_DISTRIBUTER:-yes}"
START_WORKFLOW_EVENT_ROUTER="${START_WORKFLOW_EVENT_ROUTER:-yes}"
START_WORKFLOW_NODE_EXECUTOR="${START_WORKFLOW_NODE_EXECUTOR:-yes}"
START_WORKFLOW_NODE_QUEUE_WORKER="${START_WORKFLOW_NODE_QUEUE_WORKER:-yes}"
START_NODE_REQUEST_WORKER="${START_NODE_REQUEST_WORKER:-yes}"
START_WEBHOOK_PROVISIONER="${START_WEBHOOK_PROVISIONER:-yes}"
START_WEBHOOK_CLEANUP_WORKER="${START_WEBHOOK_CLEANUP_WORKER:-yes}"
START_WORKFLOW_CLEANUP_WORKER="${START_WORKFLOW_CLEANUP_WORKER:-yes}"
NO_ENCRYPTION="${NO_ENCRYPTION:-yes}"

# Load existing values if the file exists
if [ -f "${ENV_FILE}" ]; then
  # Source the env file to load persisted values
  set -a
  source "${ENV_FILE}"
  set +a
fi

# Generate random secrets if they don't exist
if [ -z "${ENCRYPTION_KEY:-}" ] || [ "${ENCRYPTION_KEY}" = "1234567890abcdefghijklmnopqrstuv" ]; then
  ENCRYPTION_KEY=$(generate_secret)
fi
if [ -z "${JWT_SECRET:-}" ] || [ "${JWT_SECRET}" = "1234567890abcdefghijklmnopqrstuv" ]; then
  JWT_SECRET=$(generate_secret)
fi
if [ -z "${SESSION_SECRET:-}" ] || [ "${SESSION_SECRET}" = "1234567890abcdefghijklmnopqrstuv" ]; then
  SESSION_SECRET=$(generate_secret)
fi

# Generate random subdomain for localtunnel if it doesn't exist
# Format: superplane-local-{random}
if [ -z "${LOCALTUNNEL_SUBDOMAIN:-}" ]; then
  # Generate a random 8-character hex string for the subdomain
  RANDOM_SUFFIX=$(head -c 4 /dev/urandom | od -An -tx1 | tr -d ' \n' | head -c 8)
  LOCALTUNNEL_SUBDOMAIN="superplane-local-${RANDOM_SUFFIX}"
fi

# Ensure directory exists
mkdir -p "$(dirname "${ENV_FILE}")"

# Save all environment variables to file for persistence (with export statements so it can be sourced)
cat > "${ENV_FILE}" <<EOF
export PGDATA="${PGDATA}"
export DB_HOST="${DB_HOST}"
export DB_PORT="${DB_PORT}"
export DB_NAME="${DB_NAME}"
export DB_USERNAME="${DB_USERNAME}"
export DB_PASSWORD="${DB_PASSWORD}"
export POSTGRES_DB_SSL="${POSTGRES_DB_SSL}"
export APPLICATION_NAME="${APPLICATION_NAME}"
export BASE_URL="${BASE_URL}"
export WEBHOOKS_BASE_URL="${WEBHOOKS_BASE_URL}"
export RABBITMQ_URL="${RABBITMQ_URL}"
export SWAGGER_BASE_PATH="${SWAGGER_BASE_PATH}"
export RBAC_MODEL_PATH="${RBAC_MODEL_PATH}"
export RBAC_ORG_POLICY_PATH="${RBAC_ORG_POLICY_PATH}"
export TEMPLATE_DIR="${TEMPLATE_DIR}"
export WEB_BASE_PATH="${WEB_BASE_PATH}"
export PUBLIC_API_BASE_PATH="${PUBLIC_API_BASE_PATH}"
export OWNER_SETUP_ENABLED="${OWNER_SETUP_ENABLED}"
export LOCALTUNNEL_ENABLED="${LOCALTUNNEL_ENABLED}"
export LOCALTUNNEL_SUBDOMAIN="${LOCALTUNNEL_SUBDOMAIN}"
export START_PUBLIC_API="${START_PUBLIC_API}"
export START_INTERNAL_API="${START_INTERNAL_API}"
export START_GRPC_GATEWAY="${START_GRPC_GATEWAY}"
export START_CONSUMERS="${START_CONSUMERS}"
export START_WEB_SERVER="${START_WEB_SERVER}"
export START_EVENT_DISTRIBUTER="${START_EVENT_DISTRIBUTER}"
export START_WORKFLOW_EVENT_ROUTER="${START_WORKFLOW_EVENT_ROUTER}"
export START_WORKFLOW_NODE_EXECUTOR="${START_WORKFLOW_NODE_EXECUTOR}"
export START_WORKFLOW_NODE_QUEUE_WORKER="${START_WORKFLOW_NODE_QUEUE_WORKER}"
export START_NODE_REQUEST_WORKER="${START_NODE_REQUEST_WORKER}"
export START_WEBHOOK_PROVISIONER="${START_WEBHOOK_PROVISIONER}"
export START_WEBHOOK_CLEANUP_WORKER="${START_WEBHOOK_CLEANUP_WORKER}"
export START_WORKFLOW_CLEANUP_WORKER="${START_WORKFLOW_CLEANUP_WORKER}"
export ENCRYPTION_KEY="${ENCRYPTION_KEY}"
export JWT_SECRET="${JWT_SECRET}"
export SESSION_SECRET="${SESSION_SECRET}"
export NO_ENCRYPTION="${NO_ENCRYPTION}"
EOF
chmod 600 "${ENV_FILE}"

