#!/usr/bin/env bash

set -euo pipefail

if [ "$#" -ne 2 ]; then
  echo "Usage: release/superplane-single-host-tarball/smoke-test.sh <tarball> <app-image>" >&2
  exit 1
fi

TARBALL="$1"
APP_IMAGE="$2"
install_dir=""
warm_runner_id=""
smoke_project="${SUPERPLANE_SMOKE_PROJECT:-superplane-smoke-$$}"

if [[ ! "${smoke_project}" =~ ^[a-z0-9][a-z0-9_-]*$ ]]; then
  echo "Error: SUPERPLANE_SMOKE_PROJECT must contain lowercase letters, numbers, hyphens, or underscores" >&2
  exit 1
fi

cleanup() {
  if [[ "${SUPERPLANE_SMOKE_KEEP:-no}" == "yes" ]]; then
    if [[ -n "${install_dir}" ]]; then
      echo "Keeping the installation in ${install_dir}."
    fi
    return 0
  fi
  if [[ -n "${install_dir}" && -f "${install_dir}/docker-compose.yml" ]]; then
    if [[ -n "${warm_runner_id}" ]]; then
      docker rm -f "${warm_runner_id}" >/dev/null 2>&1 || true
    fi
    docker compose -f "${install_dir}/docker-compose.yml" down -v --remove-orphans >/dev/null 2>&1 || true
    rm -rf "${install_dir}"
  fi
}
trap cleanup EXIT

if [[ ! -f "${TARBALL}" ]]; then
  echo "Error: tarball does not exist: ${TARBALL}" >&2
  exit 1
fi
if ! docker image inspect "${APP_IMAGE}" >/dev/null 2>&1; then
  echo "Error: app image is not available locally: ${APP_IMAGE}" >&2
  exit 1
fi

install_dir="$(mktemp -d)"
tar -C "${install_dir}" -xzf "${TARBALL}"
install_dir="${install_dir}/superplane"
cd "${install_dir}"

perl -i -pe "s/^name: superplane\$/name: ${smoke_project}/" docker-compose.yml
perl -i -pe "s/network: superplane_default/network: ${smoke_project}_default/" install.sh

expected_image="$(awk '/image: ghcr.io\/superplanehq\/superplane:/ { print $2; exit }' docker-compose.yml)"
if [[ -z "${expected_image}" ]]; then
  echo "Error: compose file has no SuperPlane image" >&2
  exit 1
fi
if [[ "${APP_IMAGE}" != "${expected_image}" ]]; then
  docker tag "${APP_IMAGE}" "${expected_image}"
fi

echo "Pulling dependency images..."
docker compose pull db rabbitmq caddy
fleet_manager_image="$(awk '/image: ghcr.io\/superplanehq\/fleet-manager:/ { print $2; exit }' docker-compose.yml)"
docker pull "${fleet_manager_image}"
runner_version="$(awk -F'"' '/^RUNNER_VERSION=/ { print $2; exit }' install.sh)"
docker pull "ghcr.io/superplanehq/runner:${runner_version}"

echo "Installing..."
SUPERPLANE_DOMAIN=localhost SUPERPLANE_SKIP_PULL=yes ./install.sh

secret_value() {
  local key="$1"
  grep -E "^${key}=" superplane.env | head -n 1 | cut -d= -f2-
}

snapshot_secrets() {
  local key
  for key in ENCRYPTION_MASTER_KEYS JWT_SECRET DB_PASSWORD BLOB_STORAGE_SIGNING_KEY RABBITMQ_DEFAULT_PASS; do
    printf '%s=%s\n' "${key}" "$(secret_value "${key}")"
  done
}

before="$(snapshot_secrets)"

wait_for_health() {
  local _
  for _ in $(seq 1 60); do
    if curl -kfsS https://localhost/health >/dev/null 2>&1; then
      return 0
    fi
    sleep 5
  done
  echo "Error: https://localhost/health did not become ready" >&2
  docker compose ps >&2 || true
  docker compose logs --tail 80 app >&2 || true
  return 1
}

wait_for_health

fleet_manager_token="$(secret_value FLEET_MANAGER_API_TOKEN)"
fleet_ready="no"
for _ in $(seq 1 30); do
  fleets="$(curl -kfsS -H "Authorization: Bearer ${fleet_manager_token}" https://localhost/admin/api/installation/fleets 2>/dev/null || true)"
  if printf '%s\n' "${fleets}" | grep -q 'e1-large-'; then
    fleet_ready="yes"
    break
  fi
  sleep 2
done
if [[ "${fleet_ready}" != "yes" ]]; then
  echo "Error: Fleet Manager did not synchronize the local fleet" >&2
  docker compose logs --tail 80 fleet-manager >&2 || true
  exit 1
fi

warm_runner_ready="no"
runner_network="$(awk '$1 == "network:" { print $2; exit }' fleet-manager.yaml)"
for _ in $(seq 1 30); do
  warm_runner_id="$(docker ps \
    --filter label=superplane.managed-runner=true \
    --filter "network=${runner_network}" \
    --format '{{.ID}}' | head -n 1)"
  if [[ -n "${warm_runner_id}" ]]; then
    warm_runner_ready="yes"
    break
  fi
  sleep 2
done
if [[ "${warm_runner_ready}" != "yes" ]]; then
  echo "Error: Fleet Manager did not start the warm runner" >&2
  docker compose logs --tail 80 fleet-manager >&2 || true
  exit 1
fi

http_status="$(curl -s -o /dev/null -w '%{http_code}' http://localhost/health || true)"
case "${http_status}" in
  301 | 302 | 307 | 308) ;;
  *)
    echo "Error: HTTP did not redirect to HTTPS (status ${http_status})" >&2
    exit 1
    ;;
esac

echo "Re-running the installer..."
SUPERPLANE_DOMAIN=localhost SUPERPLANE_SKIP_PULL=yes ./install.sh
after="$(snapshot_secrets)"
if [[ "${before}" != "${after}" ]]; then
  echo "Error: installer regenerated secrets" >&2
  exit 1
fi

echo "Restarting services..."
docker compose restart
wait_for_health

echo "Stopping the app process..."
# docker kill is a manual stop, so the restart policy does not apply.
# PID 1 is the entrypoint shell, and a signal to that process does not stop it.
# Kill the server. The entrypoint exits and Docker applies restart: always.
app_id="$(docker compose ps -aq app)"
restarts_before="$(docker inspect --format '{{.RestartCount}}' "${app_id}")"
# The release image does not install procps, so find the server in /proc.
docker compose exec -T app bash -c 'pid=""; for d in /proc/[0-9]*; do cmd=$(tr "\0" " " < "$d/cmdline" 2>/dev/null || true); case "$cmd" in /app/build/superplane*) pid="${d#/proc/}"; break;; esac; done; test -n "$pid"; kill -9 "$pid"' || true
restarted="no"
for _ in $(seq 1 30); do
  restarts_after="$(docker inspect --format '{{.RestartCount}}' "${app_id}")"
  if [[ "${restarts_after}" -gt "${restarts_before}" ]]; then
    restarted="yes"
    break
  fi
  sleep 1
done
if [[ "${restarted}" != "yes" ]]; then
  echo "Error: app container did not restart" >&2
  exit 1
fi
wait_for_health

env_mode="$(stat -c %a superplane.env 2>/dev/null || stat -f %OLp superplane.env)"
if [[ "${env_mode}" != "600" ]]; then
  echo "Error: superplane.env mode is ${env_mode}" >&2
  exit 1
fi

for port in 5672 15672; do
  published="$(docker compose port rabbitmq "${port}" 2>/dev/null || true)"
  # An unpublished port is empty or ":0". A host binding contains a non-zero port.
  if [[ "${published}" =~ :[1-9] ]]; then
    echo "Error: RabbitMQ port ${port} is published on the host (${published})" >&2
    exit 1
  fi
done

users=""
rabbit_ready="no"
for _ in $(seq 1 30); do
  if users="$(docker compose exec -T rabbitmq rabbitmqctl list_users 2>/dev/null)"; then
    rabbit_ready="yes"
    break
  fi
  sleep 2
done
if [[ "${rabbit_ready}" != "yes" ]]; then
  echo "Error: RabbitMQ did not accept rabbitmqctl" >&2
  exit 1
fi
printf '%s\n' "${users}" | awk 'NR > 1 { print $1 }' | grep -qx superplane
if printf '%s\n' "${users}" | awk 'NR > 1 { print $1 }' | grep -qx guest; then
  echo "Error: RabbitMQ still has the guest user" >&2
  exit 1
fi

while read -r container_id; do
  [[ -n "${container_id}" ]] || continue
  max_size="$(docker inspect --format '{{index .HostConfig.LogConfig.Config "max-size"}}' "${container_id}")"
  if [[ "${max_size}" != "50m" ]]; then
    echo "Error: container ${container_id} does not limit logs to 50m" >&2
    exit 1
  fi
done < <(docker compose ps -q)

if ! curl -kfsS https://localhost/health >/dev/null; then
  echo "Error: health check failed after lifecycle tests" >&2
  exit 1
fi

echo "Single-host smoke test passed."
