#!/usr/bin/env bash

set -euo pipefail

cd "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

token="${1:-${INSTALLATION_ADMIN_TOKEN:-}}"
if [[ -z "${token}" ]]; then
  echo "Usage: ./enable-runners.sh <installation_admin_token>" >&2
  exit 1
fi
case "${token}" in
  *$'\n'* | *$'\r'* | *" "*)
    echo "Installation admin token must be one line." >&2
    exit 1
    ;;
esac

if [[ ! -f fleet-manager.yaml ]]; then
  echo "fleet-manager.yaml is missing. Run ./install.sh first." >&2
  exit 1
fi
if ! command -v docker >/dev/null 2>&1 || ! docker compose version >/dev/null 2>&1; then
  echo "Docker Compose v2 is required." >&2
  exit 1
fi

umask 077
printf 'INSTALLATION_ADMIN_TOKEN=%s\n' "${token}" >fleet-manager.env
chmod 600 fleet-manager.env

runner_image="$(awk '$1 == "image:" && $2 ~ /^ghcr.io\/superplanehq\/runner:/ { print $2; exit }' fleet-manager.yaml)"
if [[ -z "${runner_image}" ]]; then
  echo "Could not find the runner image in fleet-manager.yaml." >&2
  exit 1
fi

if [[ "${SUPERPLANE_SKIP_PULL:-no}" != "yes" ]]; then
  echo "Pulling Fleet Manager and the runner image..."
  docker compose --profile runners pull fleet-manager
  docker pull "${runner_image}"
fi

echo "Starting Fleet Manager..."
docker compose --profile runners up --wait --detach fleet-manager

echo ""
echo "Fleet Manager is running. It starts a runner container for each task."
echo "Runner image: ${runner_image}"
