#!/usr/bin/env bash

set -euo pipefail

if [ "$#" -ne 2 ]; then
  echo "Usage: release/superplane/publish-manifest.sh <version> <source-sha>" >&2
  exit 1
fi

VERSION="$1"
SOURCE_SHA="$2"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
IMAGE_REPOSITORY="${SUPERPLANE_IMAGE_REPO:-ghcr.io/superplanehq/superplane}"
IMAGE="${IMAGE_REPOSITORY}:${VERSION}"
SOURCES=("${IMAGE}-amd64" "${IMAGE}-arm64")

node "${SCRIPT_DIR}/../component-release.js" validate-version "${VERSION}" >/dev/null
if [[ ! "${SOURCE_SHA}" =~ ^[0-9a-f]{40}$ ]]; then
  echo "Error: source SHA must be a full lowercase Git SHA" >&2
  exit 1
fi

inspect_error="$(mktemp)"
trap 'rm -f "${inspect_error}"' EXIT
if docker buildx imagetools inspect --raw "${IMAGE}" >/dev/null 2>"${inspect_error}"; then
  echo "* Reusing existing image index ${IMAGE}"
elif grep -Eqi 'manifest unknown|name unknown|not found' "${inspect_error}"; then
  docker buildx imagetools create --tag "${IMAGE}" "${SOURCES[@]}"
else
  echo "Error: could not determine whether ${IMAGE} exists" >&2
  cat "${inspect_error}" >&2
  exit 1
fi

bash "${SCRIPT_DIR}/verify-image.sh" \
  "${IMAGE}" "${VERSION}" "${SOURCE_SHA}" "${SOURCES[@]}"

digest="$(docker buildx imagetools inspect "${IMAGE}" | awk '$1 == "Digest:" {print $2; exit}')"
if [[ ! "${digest}" =~ ^sha256:[0-9a-f]{64}$ ]]; then
  echo "Error: could not read the published image digest" >&2
  exit 1
fi
printf '%s\n' "${digest}"
