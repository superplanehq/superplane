#!/usr/bin/env bash

set -euo pipefail

cd "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

#
# The logo is generated using:
# script -q /dev/null npx oh-my-logo "SuperPlane" purple --filled > release/superplane-single-host-tarball/templates/superplane-logo.txt
#

cat "${PWD}/superplane-logo.txt"

ENV_FILE="superplane.env"
RUNNER_VERSION="__RUNNER_VERSION__"
FLEET_MANAGER_VERSION="__FLEET_MANAGER_VERSION__"
BLOB_PATH="/var/lib/superplane/blobs"
RUNNER_LOG_PATH="/var/lib/superplane/runner-active-logs"

echo "Running SuperPlane single-host installation."
echo ""

env_has() {
  local key="$1"
  grep -Eq "^${key}=" "${ENV_FILE}"
}

env_value() {
  local key="$1"
  grep -E "^${key}=" "${ENV_FILE}" | head -n 1 | cut -d= -f2-
}

ensure_key() {
  local key="$1"
  local value="$2"
  if env_has "${key}"; then
    return 0
  fi
  printf '%s=%s\n' "${key}" "${value}" >>"${ENV_FILE}"
}

set_key() {
  local key="$1"
  local value="$2"
  local temporary
  if ! env_has "${key}"; then
    printf '%s=%s\n' "${key}" "${value}" >>"${ENV_FILE}"
    return 0
  fi
  temporary="$(mktemp)"
  awk -v key="${key}" -v value="${value}" '
    index($0, key "=") == 1 { print key "=" value; next }
    { print }
  ' "${ENV_FILE}" >"${temporary}"
  mv "${temporary}" "${ENV_FILE}"
}

random_hex() {
  local bytes="$1"
  local raw
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -hex "${bytes}"
    return 0
  fi
  raw="$(head -c "$((bytes * 8))" /dev/urandom | tr -dc 'a-f0-9' || true)"
  printf '%s\n' "${raw:0:$((bytes * 2))}"
}

random_base64() {
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -base64 32 | tr -d '\n'
    return 0
  fi
  head -c 32 /dev/urandom | base64 | tr -d '\n'
}

port_in_use() {
  local port="$1"
  if command -v ss >/dev/null 2>&1; then
    ss -ltn | awk '{print $4}' | grep -Eq ":${port}$"
    return $?
  fi
  (echo >/dev/tcp/127.0.0.1/"${port}") >/dev/null 2>&1
}

require_docker_compose() {
  if ! command -v docker >/dev/null 2>&1; then
    echo "Docker is required. Install Docker Engine and the Compose plugin." >&2
    exit 1
  fi
  if ! docker compose version >/dev/null 2>&1; then
    echo "Docker Compose v2 is required. Install the docker compose plugin." >&2
    exit 1
  fi
}

write_fleet_manager_config() {
  local architecture fleet_id
  case "$(uname -m)" in
    aarch64 | arm64)
      architecture="arm64"
      fleet_id="e1-large-arm64"
      ;;
    x86_64 | amd64)
      architecture="amd64"
      fleet_id="e1-large-amd64"
      ;;
    *)
      echo "Unsupported architecture: $(uname -m)" >&2
      exit 1
      ;;
  esac

  cat >fleet-manager.yaml <<EOF
id: single-vm
superplaneUrl: http://app:8000
reconciliation:
  pollTimeout: 30s
  errorRetryInterval: 10s
http:
  requestTimeout: 60s
fleets:
  - id: ${fleet_id}
    provider: docker
    runnerVersion: ${RUNNER_VERSION}
    maxCapacity: 2
    spec:
      operatingSystem: linux
      architecture: ${architecture}
      cpuMillicores: 2000
      memoryMb: 8192
      diskGb: 30
      capabilities: []
    docker:
      image: ghcr.io/superplanehq/runner:${RUNNER_VERSION}
      architecture: ${architecture}
      runnerApiUrl: http://app:8000
      network: superplane_default
EOF
}

write_caddyfile() {
  local domain="$1"
  if [[ "${domain}" == "localhost" || "${domain}" == "127.0.0.1" ]]; then
    cat >Caddyfile <<EOF
${domain} {
  tls internal
  reverse_proxy app:8000
}
EOF
    return 0
  fi
  if [[ -f Caddyfile ]] && ! grep -q 'tls internal' Caddyfile; then
    return 0
  fi
  cat >Caddyfile <<'EOF'
{$SUPERPLANE_DOMAIN} {
  reverse_proxy app:8000
}
EOF
}

write_rabbitmq_env() {
  umask 077
  cat >rabbitmq.env <<EOF
RABBITMQ_DEFAULT_USER=$(env_value RABBITMQ_DEFAULT_USER)
RABBITMQ_DEFAULT_PASS=$(env_value RABBITMQ_DEFAULT_PASS)
EOF
  chmod 600 rabbitmq.env
}

ensure_oidc_keys() {
  local oidc_dir="${PWD}/oidc"
  local existing_key key_file error_file
  mkdir -p "${oidc_dir}"
  existing_key="$(find "${oidc_dir}" -type f 2>/dev/null | head -n 1 || true)"
  if [[ -n "${existing_key}" ]]; then
    return 0
  fi
  if ! command -v openssl >/dev/null 2>&1; then
    echo "openssl is required to generate OIDC keys." >&2
    exit 1
  fi
  key_file="${oidc_dir}/$(date -u +%s).pem"
  error_file="$(mktemp)"
  if ! openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:4096 -out "${key_file}" 2>"${error_file}"; then
    cat "${error_file}" >&2
    rm -f "${error_file}"
    exit 1
  fi
  rm -f "${error_file}"
  chmod 644 "${key_file}"
  chmod 755 "${oidc_dir}"
}

write_new_env() {
  local domain="$1"
  local base_url="https://${domain}"
  local encryption_key encryption_master_keys jwt_secret db_password rabbit_password signing_key
  encryption_key="$(random_hex 16)"
  encryption_master_keys="env://host-v1:$(random_base64)"
  jwt_secret="$(random_hex 32)"
  db_password="$(random_hex 16)"
  rabbit_password="$(random_hex 16)"
  signing_key="$(random_hex 32)"

  umask 077
  cat >"${ENV_FILE}" <<EOF
APP_ENV=production
APPLICATION_NAME=superplane

BASE_URL=${base_url}
WEBHOOKS_BASE_URL=${base_url}
SUPERPLANE_DOMAIN=${domain}
PUBLIC_API_BASE_PATH=/api/v1
WEB_BASE_PATH=
REDIRECT_HTTP_REQUESTS=yes

DB_HOST=db
DB_PORT=5432
DB_NAME=superplane
DB_USERNAME=postgres
DB_PASSWORD=${db_password}
POSTGRES_DB=superplane
POSTGRES_PASSWORD=${db_password}
POSTGRES_DB_SSL=false

RABBITMQ_DEFAULT_USER=superplane
RABBITMQ_DEFAULT_PASS=${rabbit_password}
RABBITMQ_URL=amqp://superplane:${rabbit_password}@rabbitmq:5672

BLOB_STORAGE_PROVIDER=filesystem
BLOB_STORAGE_LOCAL_PATH=${BLOB_PATH}
BLOB_STORAGE_SIGNING_KEY=${signing_key}

SWAGGER_BASE_PATH=/app/api/swagger
RBAC_MODEL_PATH=/app/rbac/rbac_model.conf
RBAC_ORG_POLICY_PATH=/app/rbac/rbac_org_policy.csv
TEMPLATE_DIR=/app/templates

ENCRYPTION_KEY=${encryption_key}
ENCRYPTION_MASTER_KEYS=${encryption_master_keys}
JWT_SECRET=${jwt_secret}
OIDC_KEYS_PATH=/app/oidc-keys

OWNER_SETUP_ENABLED=yes
BLOCK_SIGNUP=yes
ENABLE_PASSWORD_LOGIN=yes

START_PUBLIC_API=yes
START_WEB_SERVER=yes
START_EVENT_DISTRIBUTER=yes
START_CONSUMERS=yes
START_EVENT_ROUTER=yes
START_RUN_FINALIZER=yes
START_RUN_INITIALIZER=yes
START_NODE_EXECUTOR=yes
START_EXECUTION_TERMINATOR=yes
START_NODE_QUEUE_WORKER=yes
START_NODE_REQUEST_WORKER=yes
START_APP_MESSAGE_WORKER=yes
START_INTEGRATION_REQUEST_WORKER=yes
START_WEBHOOK_PROVISIONER=yes
START_WEBHOOK_CLEANUP_WORKER=yes
START_INTEGRATION_CLEANUP_WORKER=yes
START_CANVAS_CLEANUP_WORKER=yes
START_NODE_REQUEST_CLEANUP_WORKER=yes
START_ORGANIZATION_CLEANUP_WORKER=yes
START_FACTORY_CLEANUP_WORKER=yes
START_FILE_CLEANUP_WORKER=yes
START_FACTORY_VELOCITY_SYNC_WORKER=yes
START_PRICE_BOOK_SYNC_WORKER=yes
START_PLANNING_SESSION_CLEANUP_WORKER=yes
START_EVENT_RETENTION_WORKER=yes
START_RUNNER_API=yes
START_RUNNER_LOG_COMPACTOR=yes
START_RUNNER_CLEANUP_WORKER=yes
RUNNER_ACTIVE_LOG_FS_PATH=${RUNNER_LOG_PATH}
RUNNER_STABLE_VERSION=${RUNNER_VERSION}

SENTRY_DSN=
SENTRY_ENVIRONMENT=single-host

SUPERPLANE_BEACON_ENABLED=yes
SUPERPLANE_INSTALLATION_TYPE=single-host
OTEL_ENABLED=no

# To enable email invitations, set:
# RESEND_API_KEY=your_resend_api_key
# EMAIL_FROM_NAME=SuperPlane
# EMAIL_FROM_ADDRESS=noreply@example.com
EOF
  chmod 600 "${ENV_FILE}"
}

upgrade_env() {
  local domain="$1"
  local base_url="https://${domain}"
  local rabbit_password signing_key

  set_key SUPERPLANE_DOMAIN "${domain}"
  set_key BASE_URL "${base_url}"
  set_key WEBHOOKS_BASE_URL "${base_url}"
  ensure_key REDIRECT_HTTP_REQUESTS yes
  ensure_key POSTGRES_DB superplane
  ensure_key POSTGRES_DB_SSL false
  ensure_key BLOB_STORAGE_PROVIDER filesystem
  ensure_key BLOB_STORAGE_LOCAL_PATH "${BLOB_PATH}"
  if ! env_has BLOB_STORAGE_SIGNING_KEY; then
    signing_key="$(random_hex 32)"
    ensure_key BLOB_STORAGE_SIGNING_KEY "${signing_key}"
  fi
  if ! env_has RABBITMQ_DEFAULT_PASS || env_value RABBITMQ_URL | grep -q 'guest:guest'; then
    if env_has RABBITMQ_DEFAULT_PASS; then
      rabbit_password="$(env_value RABBITMQ_DEFAULT_PASS)"
    else
      rabbit_password="$(random_hex 16)"
      ensure_key RABBITMQ_DEFAULT_USER superplane
      ensure_key RABBITMQ_DEFAULT_PASS "${rabbit_password}"
    fi
    ensure_key RABBITMQ_DEFAULT_USER superplane
    set_key RABBITMQ_URL "amqp://superplane:${rabbit_password}@rabbitmq:5672"
  fi
  ensure_key START_ORGANIZATION_CLEANUP_WORKER yes
  ensure_key START_FACTORY_CLEANUP_WORKER yes
  ensure_key START_FILE_CLEANUP_WORKER yes
  ensure_key START_FACTORY_VELOCITY_SYNC_WORKER yes
  ensure_key START_PRICE_BOOK_SYNC_WORKER yes
  ensure_key START_PLANNING_SESSION_CLEANUP_WORKER yes
  ensure_key START_EVENT_RETENTION_WORKER yes
  ensure_key START_RUNNER_API yes
  ensure_key START_RUNNER_LOG_COMPACTOR yes
  ensure_key START_RUNNER_CLEANUP_WORKER yes
  ensure_key RUNNER_ACTIVE_LOG_FS_PATH "${RUNNER_LOG_PATH}"
  set_key RUNNER_STABLE_VERSION "${RUNNER_VERSION}"
  chmod 600 "${ENV_FILE}"
}

DOMAIN="${SUPERPLANE_DOMAIN:-}"
existing_env="no"
if [[ -f "${ENV_FILE}" ]]; then
  existing_env="yes"
fi
if [[ -z "${DOMAIN}" && "${existing_env}" == "yes" ]]; then
  DOMAIN="$(env_value SUPERPLANE_DOMAIN)"
fi
if [[ -z "${DOMAIN}" ]]; then
  while :; do
    read -rp "Domain for SuperPlane (e.g. superplane.example.com): " DOMAIN
    if [[ -n "${DOMAIN}" ]]; then
      break
    fi
    echo "Domain is required. Please enter a value."
  done
fi

SANITIZED_DOMAIN="$(echo "${DOMAIN}" | sed -E 's#^https?://##' | cut -d'/' -f1)"
if [[ -z "${SANITIZED_DOMAIN}" ]]; then
  echo "Domain is required." >&2
  exit 1
fi

echo ""
if [[ "${existing_env}" == "yes" ]]; then
  echo "Keeping existing secrets in ${ENV_FILE}."
  upgrade_env "${SANITIZED_DOMAIN}"
else
  echo "Generating secrets..."
  write_new_env "${SANITIZED_DOMAIN}"
fi

ensure_oidc_keys
write_fleet_manager_config
write_caddyfile "${SANITIZED_DOMAIN}"
write_rabbitmq_env

if [[ ! -f fleet-manager.env ]]; then
  umask 077
  printf '%s\n' "# Open setup, select This VM, and run the command from that page." >fleet-manager.env
  chmod 600 fleet-manager.env
fi

export OIDC_KEYS_HOST_PATH="${PWD}/oidc"
export OIDC_KEYS_CONTAINER_PATH="/app/oidc-keys"

echo ""
echo "Configuration written to ${ENV_FILE}."
echo "Runner fleet configuration written to fleet-manager.yaml."
echo ""

if [[ "${SUPERPLANE_CONFIG_ONLY:-no}" == "yes" ]]; then
  echo "Configuration only. Not starting containers."
  exit 0
fi

require_docker_compose

if [[ "${existing_env}" == "no" ]]; then
  for port in 80 443; do
    if port_in_use "${port}"; then
      echo "Port ${port} is already in use. Free it before installing SuperPlane." >&2
      exit 1
    fi
  done
fi

if [[ "${SUPERPLANE_SKIP_PULL:-no}" == "yes" ]]; then
  echo "Skipping image pull."
else
  echo "Pulling Docker images (this may take a while)..."
  docker compose pull --quiet db rabbitmq caddy app
fi

echo "Running docker compose up --wait --detach..."
docker compose up --wait --detach

echo ""
echo "SuperPlane is starting via docker compose."
echo "Visit https://${SANITIZED_DOMAIN} in your browser."
echo "Runners use runner ${RUNNER_VERSION} and Fleet Manager ${FLEET_MANAGER_VERSION}."
echo "To start runners, open setup, select This VM, and run the command from that page."
