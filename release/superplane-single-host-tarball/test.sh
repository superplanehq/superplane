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
capacity_dir=""
uninstall_dir=""
trap 'rm -rf "${work_dir}" "${guest_dir}" "${capacity_dir}" "${uninstall_dir}"' EXIT
tar -C "${work_dir}" -xzf "${REPO_ROOT}/build/superplane-single-host-tarball-${VERSION}/superplane-single-host.tar.gz"
cd "${work_dir}/superplane"

if grep -R -E '__SUPERPLANE_VERSION__|__RUNNER_VERSION__|__FLEET_MANAGER_VERSION__' . >/dev/null; then
  echo "Error: release placeholders were not replaced" >&2
  exit 1
fi
grep -q "ghcr.io/superplanehq/superplane:${VERSION}" docker-compose.yml
grep -q "ghcr.io/superplanehq/fleet-manager:${FLEET_MANAGER_VERSION}-docker" docker-compose.yml
test ! -e enable-runners.sh
test -x uninstall.sh
./uninstall.sh --help >/dev/null

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
  SUPERPLANE_BILLING_ENABLED
  START_ORGANIZATION_CLEANUP_WORKER
  START_FILE_CLEANUP_WORKER
  START_EVENT_RETENTION_WORKER
  ENCRYPTION_MASTER_KEYS
  JWT_SECRET
  FLEET_MANAGER_API_TOKEN
  SUPERPLANE_RUNNER_MAX_CAPACITY
  SUPERPLANE_RUNNER_WARM_CAPACITY
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
if ! grep -Eq '^SUPERPLANE_BILLING_ENABLED=no$' superplane.env; then
  echo "Error: single-host billing must be off" >&2
  exit 1
fi

env_mode="$(stat -c %a superplane.env 2>/dev/null || stat -f %OLp superplane.env)"
test "${env_mode}" = "600"
rabbit_mode="$(stat -c %a rabbitmq.env 2>/dev/null || stat -f %OLp rabbitmq.env)"
test "${rabbit_mode}" = "600"
grep -Eq '^RABBITMQ_DEFAULT_USER=superplane$' rabbitmq.env
if grep -Eq 'JWT_SECRET|ENCRYPTION_MASTER_KEYS|DB_PASSWORD' rabbitmq.env; then
  echo "Error: rabbitmq.env contains app secrets" >&2
  exit 1
fi
test -f fleet-manager.yaml
grep -q "runnerVersion: ${RUNNER_VERSION}" fleet-manager.yaml
grep -q "network: superplane_default" fleet-manager.yaml
grep -Eq '^    warmCapacity: 1$' fleet-manager.yaml
grep -Eq '^    maxCapacity: 5$' fleet-manager.yaml
fleet_token="$(grep '^FLEET_MANAGER_API_TOKEN=' superplane.env | cut -d= -f2-)"
test -n "${fleet_token}"
test "$(grep '^FLEET_MANAGER_API_TOKEN=' fleet-manager.env | cut -d= -f2-)" = "${fleet_token}"
test "$(grep '^INSTALLATION_ADMIN_TOKEN=' fleet-manager.env | cut -d= -f2-)" = "${fleet_token}"
fleet_manager_mode="$(stat -c %a fleet-manager.env 2>/dev/null || stat -f %OLp fleet-manager.env)"
test "${fleet_manager_mode}" = "600"

secret_snapshot() {
  local key
  for key in ENCRYPTION_MASTER_KEYS JWT_SECRET DB_PASSWORD BLOB_STORAGE_SIGNING_KEY RABBITMQ_DEFAULT_PASS FLEET_MANAGER_API_TOKEN; do
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
grep -v '^SUPERPLANE_BILLING_ENABLED=' superplane.env >superplane.env.tmp
mv superplane.env.tmp superplane.env
chmod 600 superplane.env
SUPERPLANE_CONFIG_ONLY=yes ./install.sh >/dev/null
grep -Eq '^SUPERPLANE_BILLING_ENABLED=no$' superplane.env
upgraded="$(secret_snapshot)"
if [[ "${before}" != "${upgraded}" ]]; then
  echo "Error: upgrade changed existing secrets" >&2
  exit 1
fi

export OIDC_KEYS_HOST_PATH="${PWD}/oidc"
export OIDC_KEYS_CONTAINER_PATH="/app/oidc-keys"
docker compose config -q
rendered="$(docker compose config)"
printf '%s\n' "${rendered}" | grep -q 'fleet-manager:'
if printf '%s\n' "${rendered}" | grep -Eq 'published:.*5672|published:.*15672'; then
  echo "Error: RabbitMQ ports are published" >&2
  exit 1
fi
printf '%s\n' "${rendered}" | grep -q 'max-size: 50m'
rabbit_block="$(printf '%s\n' "${rendered}" | awk '
  $0 ~ "^  rabbitmq:" { flag = 1 }
  flag && $0 ~ "^  [a-z]" && $0 !~ "^  rabbitmq:" { flag = 0 }
  flag { print }
')"
printf '%s\n' "${rabbit_block}" | grep -q 'RABBITMQ_DEFAULT_USER'
if printf '%s\n' "${rabbit_block}" | grep -Eq 'JWT_SECRET|ENCRYPTION_MASTER_KEYS|DB_PASSWORD'; then
  echo "Error: RabbitMQ receives app secrets" >&2
  exit 1
fi

cd "${work_dir}"
SUPERPLANE_DOMAIN=localhost SUPERPLANE_CONFIG_ONLY=yes bash superplane/install.sh >/dev/null
grep -q 'tls internal' superplane/Caddyfile
SUPERPLANE_DOMAIN=superplane.example.com SUPERPLANE_CONFIG_ONLY=yes bash superplane/install.sh >/dev/null
if grep -q 'tls internal' superplane/Caddyfile; then
  echo "Error: public domain kept the local Caddyfile" >&2
  exit 1
fi
grep -q '{$SUPERPLANE_DOMAIN}' superplane/Caddyfile
cd superplane

guest_dir="$(mktemp -d)"
tar -C "${guest_dir}" -xzf "${REPO_ROOT}/build/superplane-single-host-tarball-${VERSION}/superplane-single-host.tar.gz"
cd "${guest_dir}/superplane"
SUPERPLANE_DOMAIN=superplane.example.com SUPERPLANE_CONFIG_ONLY=yes ./install.sh >/dev/null
master_keys="$(grep '^ENCRYPTION_MASTER_KEYS=' superplane.env)"
printf 'RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672\n' >>superplane.env
grep -v '^RABBITMQ_DEFAULT_PASS=' superplane.env >superplane.env.tmp
mv superplane.env.tmp superplane.env
chmod 600 superplane.env
SUPERPLANE_CONFIG_ONLY=yes ./install.sh >/dev/null
grep -Eq '^RABBITMQ_URL=amqp://superplane:' superplane.env
test "$(grep '^ENCRYPTION_MASTER_KEYS=' superplane.env)" = "${master_keys}"
if grep -Eq '^RABBITMQ_URL=amqp://guest:guest@' superplane.env; then
  echo "Error: guest RabbitMQ URL was kept" >&2
  exit 1
fi

capacity_dir="$(mktemp -d)"
tar -C "${capacity_dir}" -xzf "${REPO_ROOT}/build/superplane-single-host-tarball-${VERSION}/superplane-single-host.tar.gz"
cd "${capacity_dir}/superplane"
SUPERPLANE_DOMAIN=superplane.example.com \
  SUPERPLANE_RUNNER_MAX_CAPACITY=8 \
  SUPERPLANE_RUNNER_WARM_CAPACITY=2 \
  SUPERPLANE_CONFIG_ONLY=yes \
  ./install.sh >/dev/null
grep -Eq '^    warmCapacity: 2$' fleet-manager.yaml
grep -Eq '^    maxCapacity: 8$' fleet-manager.yaml
grep -Eq '^SUPERPLANE_RUNNER_MAX_CAPACITY=8$' superplane.env
grep -Eq '^SUPERPLANE_RUNNER_WARM_CAPACITY=2$' superplane.env
SUPERPLANE_CONFIG_ONLY=yes ./install.sh >/dev/null
grep -Eq '^    warmCapacity: 2$' fleet-manager.yaml
grep -Eq '^    maxCapacity: 8$' fleet-manager.yaml

if SUPERPLANE_RUNNER_MAX_CAPACITY=0 SUPERPLANE_CONFIG_ONLY=yes ./install.sh >/dev/null 2>&1; then
  echo "Error: zero maximum runner capacity was accepted" >&2
  exit 1
fi
if SUPERPLANE_RUNNER_MAX_CAPACITY=1 SUPERPLANE_RUNNER_WARM_CAPACITY=2 SUPERPLANE_CONFIG_ONLY=yes ./install.sh >/dev/null 2>&1; then
  echo "Error: warm runner capacity above the maximum was accepted" >&2
  exit 1
fi

printf 'INSTALLATION_ADMIN_TOKEN=legacy-admin-token\n' >fleet-manager.env
chmod 600 fleet-manager.env
SUPERPLANE_CONFIG_ONLY=yes ./install.sh >/dev/null
if [[ "$(grep -E '^(INSTALLATION_ADMIN_TOKEN|FLEET_MANAGER_API_TOKEN)=' fleet-manager.env)" != "INSTALLATION_ADMIN_TOKEN=legacy-admin-token" ]]; then
  echo "Error: installer replaced the legacy installation admin token" >&2
  exit 1
fi
legacy_mode="$(stat -c %a fleet-manager.env 2>/dev/null || stat -f %OLp fleet-manager.env)"
test "${legacy_mode}" = "600"
grep -Eq '^    warmCapacity: 2$' fleet-manager.yaml
grep -Eq '^    maxCapacity: 8$' fleet-manager.yaml

SUPERPLANE_DOMAIN=superplane.example.com \
  SUPERPLANE_RUNNER_MAX_CAPACITY=10 \
  SUPERPLANE_RUNNER_WARM_CAPACITY=08 \
  SUPERPLANE_CONFIG_ONLY=yes \
  ./install.sh >/dev/null
grep -Eq '^    warmCapacity: 8$' fleet-manager.yaml
grep -Eq '^    maxCapacity: 10$' fleet-manager.yaml
grep -Eq '^SUPERPLANE_RUNNER_WARM_CAPACITY=8$' superplane.env
if SUPERPLANE_RUNNER_MAX_CAPACITY=5 SUPERPLANE_RUNNER_WARM_CAPACITY=09 SUPERPLANE_CONFIG_ONLY=yes ./install.sh >/dev/null 2>&1; then
  echo "Error: zero-padded warm capacity above the maximum was accepted" >&2
  exit 1
fi

# Retire runners with the configured credential before physical removal.
# Substitute only Docker replies; execute the real packaged uninstall script.
uninstall_dir="$(mktemp -d)"
mkdir "${uninstall_dir}/bin" "${uninstall_dir}/installation"
cp uninstall.sh docker-compose.yml "${uninstall_dir}/installation/"
cat >"${uninstall_dir}/bin/docker" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
[[ -z "${EXPECTED_RUNNER_TOKEN}" || "$*" != *"${EXPECTED_RUNNER_TOKEN}"* ]] || {
  echo 'Credential appeared in command arguments' >&2
  exit 1
}
case "$*" in
  'compose version' | info) ;;
  'ps '*)
    case "$*" in
      *label=com.docker.compose.project=*) [[ -e removed-stack ]] || echo app ;;
      *network=* | *id=runner*) [[ -e removed-runner ]] || echo runner ;;
      *) exit 1 ;;
    esac
    ;;
  'network ls '*) [[ -e removed-stack ]] || echo test-network ;;
  'network inspect '*) echo 'superplane default' ;;
  'volume ls '*) ;;
  'inspect '*)
    case "$*" in
      *working_dir*) pwd -P ;;
      *config_files*) printf '%s/docker-compose.yml\n' "$(pwd -P)" ;;
      *'.Name'*) echo 'runner-1 test-fleet /superplane-runner-runner-1' ;;
      *superplane.runner-id*) echo 'runner-1 test-fleet' ;;
      *State.Running*) echo true ;;
      *) exit 1 ;;
    esac
    ;;
  'compose '*)
    case "$*" in
      *' exec '*)
        header="$(cat)"
        [[ "${header}" == "Authorization: Bearer ${EXPECTED_RUNNER_TOKEN}" ]] || {
          echo 'Unexpected runner API credential' >&2
          exit 1
        }
        echo retire >>events
        printf '{"runner":{"state":"terminated"}}\n200'
        ;;
      *' stop '*) ;;
      *' down '*) touch removed-stack ;;
      *) exit 1 ;;
    esac
    ;;
  'rm -f runner')
    grep -qx retire events
    touch removed-runner
    ;;
  *) echo "Unexpected Docker command: $*" >&2; exit 1 ;;
esac
EOF
chmod +x "${uninstall_dir}/bin/docker"
cd "${uninstall_dir}/installation"
for credentials in app admin manager missing; do
  rm -f removed-stack removed-runner events
  : >superplane.env
  : >fleet-manager.env
  expected=""
  case "${credentials}" in
    app)
      expected=app-fixture-token
      printf 'FLEET_MANAGER_API_TOKEN=%s\n' "${expected}" >superplane.env
      printf 'INSTALLATION_ADMIN_TOKEN=other-fixture-token\n' >fleet-manager.env
      ;;
    manager)
      expected=manager-fixture-token
      printf 'FLEET_MANAGER_API_TOKEN=%s\nINSTALLATION_ADMIN_TOKEN=other-fixture-token\n' "${expected}" >fleet-manager.env
      ;;
    admin)
      expected=admin-fixture-token
      printf 'INSTALLATION_ADMIN_TOKEN=%s\n' "${expected}" >fleet-manager.env
      ;;
  esac
  uninstall_status=0
  PATH="${uninstall_dir}/bin:${PATH}" EXPECTED_RUNNER_TOKEN="${expected}" \
    ./uninstall.sh --force --yes >result.log 2>&1 || uninstall_status=$?
  if [[ "${credentials}" == missing ]]; then
    test "${uninstall_status}" -ne 0
    grep -q 'No valid runner API token found' result.log
    test ! -e removed-runner
    test ! -e removed-stack
    test ! -e events
  else
    if [[ "${uninstall_status}" -ne 0 ]]; then
      echo "Error: uninstall failed with ${credentials} credentials" >&2
      cat result.log >&2
      exit 1
    fi
    test -e removed-runner
    test -e removed-stack
    grep -qx retire events
  fi
done

echo "Single-host tarball tests passed."
