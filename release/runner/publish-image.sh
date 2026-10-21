#!/usr/bin/env bash

set -euo pipefail

if [ "$#" -ne 3 ]; then
  echo "Usage: release/runner/publish-image.sh <version> <oci-archive> <source-sha>" >&2
  exit 1
fi

VERSION="$1"
OCI_ARCHIVE="$2"
SOURCE_SHA="$3"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
IMAGE_REPOSITORY="${RUNNER_IMAGE_REPO:-ghcr.io/superplanehq/runner}"
digest_file="$(mktemp)"
trap 'rm -f "${digest_file}"' EXIT

if [[ "${VERSION}" == sha:* ]]; then
  echo "Skipping runner image publish for SHA release ${VERSION}."
  exit 0
fi

IMAGE_REPOSITORY="${IMAGE_REPOSITORY}" \
  FLEET_MANAGER_DIGEST_OUTPUT="${digest_file}" \
  bash "${REPO_ROOT}/release/fleet-manager/publish-image.sh" \
  "${VERSION}" \
  "${OCI_ARCHIVE}"

bash "${REPO_ROOT}/release/fleet-manager/verify-image.sh" \
  "${IMAGE_REPOSITORY}:${VERSION}" \
  "$(tr -d '[:space:]' <"${digest_file}")" \
  "${VERSION}" \
  "${SOURCE_SHA}"
