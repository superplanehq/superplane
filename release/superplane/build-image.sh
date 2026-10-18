#!/usr/bin/env bash

set -euo pipefail

if [ "$#" -ne 4 ]; then
  echo "Usage: release/superplane/build-image.sh <version> <source-sha> <architecture> <output>" >&2
  exit 1
fi

VERSION="$1"
SOURCE_SHA="$2"
ARCHITECTURE="$3"
OUTPUT="$4"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
DEV_BASE_IMAGE_REPO="${DEV_BASE_IMAGE_REPO:-ghcr.io/superplanehq/superplane-dev-base}"

node "${REPO_ROOT}/release/component-release.js" validate-version "${VERSION}" >/dev/null
if [[ ! "${SOURCE_SHA}" =~ ^[0-9a-f]{40}$ ]]; then
  echo "Error: source SHA must be a full lowercase Git SHA" >&2
  exit 1
fi
case "${ARCHITECTURE}" in
  amd64 | arm64) ;;
  *)
    echo "Error: architecture must be amd64 or arm64" >&2
    exit 1
    ;;
esac

# shellcheck source=../lib/image-build-prerequisites.sh
source "${REPO_ROOT}/release/lib/image-build-prerequisites.sh"
require_release_image_build_prerequisites

if generated_release_build_inputs_missing; then
  echo "Error: generated release inputs are missing; run make dev.setup first" >&2
  exit 1
fi

mkdir -p "$(dirname "${OUTPUT}")"
test ! -e "${OUTPUT}"

cd "${REPO_ROOT}"
docker buildx build \
  --platform "linux/${ARCHITECTURE}" \
  --progress=plain \
  --provenance=mode=max \
  --build-arg "SUPERPLANE_VERSION=${VERSION}" \
  --build-arg "SOURCE_REVISION=${SOURCE_SHA}" \
  --target runner \
  --cache-from "${DEV_BASE_IMAGE_REPO}:app-latest-${ARCHITECTURE}" \
  --output "type=oci,dest=${OUTPUT}" \
  -f Dockerfile \
  .

bash "${SCRIPT_DIR}/verify-oci-archive.sh" \
  "${OUTPUT}" "${VERSION}" "${SOURCE_SHA}" "${ARCHITECTURE}"
sha256sum "${OUTPUT}" | awk '{print $1}' >"${OUTPUT}.sha256"
tar -xOf "${OUTPUT}" index.json >"${OUTPUT}.descriptor.json"
