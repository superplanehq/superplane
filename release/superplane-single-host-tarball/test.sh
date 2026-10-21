#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
VERSION="v9.9.9"
VALUES_FILE="${REPO_ROOT}/release/superplane-helm-chart/helm/values.yaml"
RUNNER_VERSION="$(awk '$1 == "runnerVersion:" { print $2; exit }' "${VALUES_FILE}")"
FLEET_MANAGER_VERSION="$(awk '$1 == "fleetManagerVersion:" { print $2; exit }' "${VALUES_FILE}")"

bash "${SCRIPT_DIR}/build.sh" "${VERSION}"

work_dir="$(mktemp -d)"
guest_dir=""
trap 'rm -rf "${work_dir}" "${guest_dir}"' EXIT
tar -C "${work_dir}" -xzf "${REPO_ROOT}/build/superplane-single-host-tarball-${VERSION}/superplane-single-host.tar.gz"
cd "${work_dir}/superplane"

if grep -R -E '__SUPERPLANE_VERSION__|__RUNNER_VERSION__|__FLEET_MANAGER_VERSION__' . >/dev/null; then
  echo "Error: release placeholders were not replaced" >&2
  exit 1
fi
grep -q "ghcr.io/superplanehq/superplane:${VERSION}" docker-compose.yml
grep -q "ghcr.io/superplanehq/fleet-manager:${FLEET_MANAGER_VERSION}-docker" docker-compose.yml

SUPERPLANE_DOMAIN=superplane.example.com SUPERPLANE_CONFIG_ONLY=yes ./install.sh >/dev/null
grep -q "image: ghcr.io/superplanehq/runner:${RUNNER_VERSION}" fleet-manager.yaml

required_keys=(
  BLOB_STORAGE_PROVIDER
  BLOB_STORAGE_LOCAL_PATH
  BLOB_STORAGE_SIGNING_KEY
  RABBITMQ_DEFAULT_USER
  RABBITMQ_DEFAULT_PASS
  RABBITMQ_URL
  REDIRECT_HTTP_REQUESTS
  START_RUNNER_API
  START_RUNNER_LOG_COMPACTOR
  START_RUNNER_CLEANUP_WORKER
  RUNNER_ACTIVE_LOG_FS_PATH
  RUNNER_STABLE_VERSION
  START_ORGANIZATION_CLEANUP_WORKER
  START_FILE_CLEANUP_WORKER
  START_EVENT_RETENTION_WORKER
  ENCRYPTION_KEY
  ENCRYPTION_MASTER_KEYS
  JWT_SECRET
)
for key in "${required_keys[@]}"; do
  if ! grep -Eq "^${key}=" superplane.env; then
    echo "Error: ${key} is missing from superplane.env" >&2
    exit 1
  fi
done
if grep -Eq '^SESSION_SECRET=' superplane.env; then
  echo "Error: SESSION_SECRET is unused and must not be written" >&2
  exit 1
fi
if grep -Eq '^RABBITMQ_URL=amqp://guest:guest@' superplane.env; then
  echo "Error: RabbitMQ still uses guest:guest" >&2
  exit 1
fi

env_mode="$(stat -c %a superplane.env 2>/dev/null || stat -f %OLp superplane.env)"
test "${env_mode}" = "600"
test -f fleet-manager.yaml
grep -q "runnerVersion: ${RUNNER_VERSION}" fleet-manager.yaml
grep -q "network: superplane_default" fleet-manager.yaml

secret_snapshot() {
  local key
  for key in ENCRYPTION_KEY ENCRYPTION_MASTER_KEYS JWT_SECRET DB_PASSWORD BLOB_STORAGE_SIGNING_KEY RABBITMQ_DEFAULT_PASS; do
    grep -E "^${key}=" superplane.env
  done
}

before="$(secret_snapshot)"
SUPERPLANE_CONFIG_ONLY=yes ./install.sh >/dev/null
after="$(secret_snapshot)"
if [[ "${before}" != "${after}" ]]; then
  echo "Error: re-running install.sh changed secrets" >&2
  exit 1
fi

grep -v '^BLOB_STORAGE_PROVIDER=' superplane.env >superplane.env.tmp
mv superplane.env.tmp superplane.env
chmod 600 superplane.env
SUPERPLANE_CONFIG_ONLY=yes ./install.sh >/dev/null
grep -Eq '^BLOB_STORAGE_PROVIDER=filesystem$' superplane.env
upgraded="$(secret_snapshot)"
if [[ "${before}" != "${upgraded}" ]]; then
  echo "Error: upgrade changed existing secrets" >&2
  exit 1
fi

export OIDC_KEYS_HOST_PATH="${PWD}/oidc"
export OIDC_KEYS_CONTAINER_PATH="/app/oidc-keys"
docker compose config -q
docker compose --profile runners config -q
rendered="$(docker compose --profile runners config)"
if printf '%s\n' "${rendered}" | grep -Eq 'published:.*5672|published:.*15672'; then
  echo "Error: RabbitMQ ports are published" >&2
  exit 1
fi
printf '%s\n' "${rendered}" | grep -q 'max-size: 50m'

guest_dir="$(mktemp -d)"
tar -C "${guest_dir}" -xzf "${REPO_ROOT}/build/superplane-single-host-tarball-${VERSION}/superplane-single-host.tar.gz"
cd "${guest_dir}/superplane"
SUPERPLANE_DOMAIN=superplane.example.com SUPERPLANE_CONFIG_ONLY=yes ./install.sh >/dev/null
old_key="$(grep '^ENCRYPTION_KEY=' superplane.env)"
printf 'RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672\n' >>superplane.env
grep -v '^RABBITMQ_DEFAULT_PASS=' superplane.env >superplane.env.tmp
mv superplane.env.tmp superplane.env
chmod 600 superplane.env
SUPERPLANE_CONFIG_ONLY=yes ./install.sh >/dev/null
grep -Eq '^RABBITMQ_URL=amqp://superplane:' superplane.env
test "$(grep '^ENCRYPTION_KEY=' superplane.env)" = "${old_key}"
if grep -Eq '^RABBITMQ_URL=amqp://guest:guest@' superplane.env; then
  echo "Error: guest RabbitMQ URL was kept" >&2
  exit 1
fi

echo "Single-host tarball tests passed."
