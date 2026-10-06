#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

if [ "${1-}" = "" ] || [ "${2-}" = "" ]; then
  echo "Usage: release/fleet-manager/build.sh <git-sha> <arch>" >&2
  exit 1
fi

GIT_SHA="$1"
ARCH="$2"

if [[ ! "${GIT_SHA}" =~ ^[0-9a-f]{40}$ ]]; then
  echo "Error: git-sha must be a full lowercase 40-character commit SHA" >&2
  exit 1
fi
if [ "${ARCH}" != "amd64" ] && [ "${ARCH}" != "arm64" ]; then
  echo "Error: arch must be amd64 or arm64" >&2
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
IMAGE_REPO="${FLEET_MANAGER_IMAGE_REPO:-ghcr.io/superplanehq/fleet-manager}"

# shellcheck source=../lib/image-build-prerequisites.sh
source "${REPO_ROOT}/release/lib/image-build-prerequisites.sh"

cd "${REPO_ROOT}"
require_docker_buildx

checked_out_sha="$(git rev-parse HEAD)"
if [ "${checked_out_sha}" != "${GIT_SHA}" ]; then
  echo "Error: checked-out commit ${checked_out_sha} does not match ${GIT_SHA}" >&2
  exit 1
fi

echo "Building Fleet Manager image ${IMAGE_REPO}:${GIT_SHA}-${ARCH}"

docker buildx build \
  --platform "linux/${ARCH}" \
  --progress=plain \
  --provenance=false \
  --push \
  --target runtime \
  -t "${IMAGE_REPO}:${GIT_SHA}-${ARCH}" \
  -f release/fleet-manager/Dockerfile \
  .
