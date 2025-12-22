#!/usr/bin/env bash

set -euo pipefail

BLUE="\033[0;34m"
LBLUE="\033[1;34m"
CLEAR="\033[0m"

#
# The logo is generated using:
# script -q /dev/null npx oh-my-logo "SuperPlane" purple --filled > release/single-host/templates/superplane-logo.txt
#

cat "$(dirname "${BASH_SOURCE[0]:-$0}")/superplane-logo.txt"

ENV_FILE="superplane.env"

echo "Running Superplane single-host installation."
echo ""

while :; do
  read -rp "1) Domain for Superplane (e.g. superplane.example.com): " DOMAIN
  if [[ -n "${DOMAIN}" ]]; then
    break
  fi
  echo "Domain is required. Please enter a value."
done

SANITIZED_DOMAIN="$(echo "${DOMAIN}" | sed -E 's#^https?://##' | cut -d'/' -f1)"
BASE_URL="https://${SANITIZED_DOMAIN}"

echo ""
read -rp "2) Configure email invitations via Resend now? (y/N): " CONFIGURE_EMAIL
CONFIGURE_EMAIL="${CONFIGURE_EMAIL:-n}"

RESEND_API_KEY=""
EMAIL_FROM_NAME="Superplane"
EMAIL_FROM_ADDRESS=""

if [[ "${CONFIGURE_EMAIL}" =~ ^[Yy]$ ]]; then
  echo ""
  read -rp "Resend API key: " RESEND_API_KEY
  read -rp "Sender name [${EMAIL_FROM_NAME}]: " EMAIL_FROM_NAME_INPUT
  EMAIL_FROM_NAME="${EMAIL_FROM_NAME_INPUT:-$EMAIL_FROM_NAME}"

  default_sender="noreply@notifications.${SANITIZED_DOMAIN}"
  read -rp "Sender email [${default_sender}]: " EMAIL_FROM_ADDRESS_INPUT
  EMAIL_FROM_ADDRESS="${EMAIL_FROM_ADDRESS_INPUT:-$default_sender}"
fi

echo ""
read -rp "3) Allow open signups (anyone can create an account)? (y/N): " ALLOW_SIGNUP_INPUT
ALLOW_SIGNUP_INPUT="${ALLOW_SIGNUP_INPUT:-n}"

if [[ "${ALLOW_SIGNUP_INPUT}" =~ ^[Yy]$ ]]; then
  BLOCK_SIGNUP="no"
else
  BLOCK_SIGNUP="yes"
fi

echo ""
echo "Generating secrets..."

if command -v openssl >/dev/null 2>&1; then
  ENCRYPTION_KEY="$(openssl rand -hex 16)"
  JWT_SECRET="$(openssl rand -hex 32)"
  SESSION_SECRET="$(openssl rand -hex 32)"
  DB_PASSWORD="$(openssl rand -hex 16)"
else
  echo "openssl not found, using /dev/urandom for secrets."
  ENCRYPTION_KEY="$(head -c 32 /dev/urandom | tr -dc 'a-f0-9' | head -c 32)"
  JWT_SECRET="$(head -c 64 /dev/urandom | tr -dc 'a-f0-9' | head -c 64)"
  SESSION_SECRET="$(head -c 64 /dev/urandom | tr -dc 'a-f0-9' | head -c 64)"
  DB_PASSWORD="$(head -c 32 /dev/urandom | tr -dc 'a-f0-9' | head -c 32)"
fi

echo ""
echo "Writing ${ENV_FILE}..."

cat > "${ENV_FILE}" <<EOF
APP_ENV=production
APPLICATION_NAME=superplane

BASE_URL=${BASE_URL}
SUPERPLANE_DOMAIN=${SANITIZED_DOMAIN}
PUBLIC_API_BASE_PATH=/api/v1
WEB_BASE_PATH=

DB_HOST=db
DB_PORT=5432
DB_NAME=superplane
DB_USERNAME=postgres
DB_PASSWORD=${DB_PASSWORD}
POSTGRES_DB_SSL=false

POSTGRES_PASSWORD=${DB_PASSWORD}

RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672

SWAGGER_BASE_PATH=/app/api/swagger
RBAC_MODEL_PATH=/app/rbac/rbac_model.conf
RBAC_ORG_POLICY_PATH=/app/rbac/rbac_org_policy.csv
TEMPLATE_DIR=/app/templates

ENCRYPTION_KEY=${ENCRYPTION_KEY}
JWT_SECRET=${JWT_SECRET}
SESSION_SECRET=${SESSION_SECRET}
NO_ENCRYPTION=no

OWNER_SETUP_ENABLED=yes
BLOCK_SIGNUP=${BLOCK_SIGNUP}

START_PUBLIC_API=yes
START_INTERNAL_API=yes
START_GRPC_GATEWAY=yes
START_CONSUMERS=yes
START_WEB_SERVER=yes
START_EVENT_DISTRIBUTER=yes
START_WORKFLOW_EVENT_ROUTER=yes
START_WORKFLOW_NODE_EXECUTOR=yes
START_WORKFLOW_NODE_QUEUE_WORKER=yes
START_NODE_REQUEST_WORKER=yes
START_WEBHOOK_PROVISIONER=yes
START_WEBHOOK_CLEANUP_WORKER=yes
START_INSTALLATION_CLEANUP_WORKER=yes
START_WORKFLOW_CLEANUP_WORKER=yes

SENTRY_DSN=
SENTRY_ENVIRONMENT=single-host

OTEL_ENABLED=no
EOF

if [[ -n "${RESEND_API_KEY}" ]]; then
  {
    echo "RESEND_API_KEY=${RESEND_API_KEY}"
    echo "EMAIL_FROM_NAME=${EMAIL_FROM_NAME}"
    echo "EMAIL_FROM_ADDRESS=${EMAIL_FROM_ADDRESS}"
  } >> "${ENV_FILE}"
else
  {
    echo "# To enable email invitations, set:"
    echo "# RESEND_API_KEY=your_resend_api_key"
    echo "# EMAIL_FROM_NAME=Superplane"
    echo "# EMAIL_FROM_ADDRESS=noreply@example.com"
  } >> "${ENV_FILE}"
fi

echo ""
echo "Configuration written to ${ENV_FILE}."
echo ""

SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]:-$0}")"

echo "Running docker compose pull..."
docker compose -f "${SCRIPT_DIR}/docker-compose.yml" pull

echo "Running docker compose up --wait --detach..."
docker compose -f "${SCRIPT_DIR}/docker-compose.yml" up --wait --detach

echo ""
echo "Superplane is starting via docker compose."
echo "Visit ${BASE_URL} in your browser."
