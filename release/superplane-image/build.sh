#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

if [ "${1-}" = "" ]; then
  echo "Usage: release/superplane-image/build.sh <version>"
  exit 1
fi

if [ "${2-}" = "" ]; then
  echo "Usage: release/superplane-image/build.sh <version> <arch>"
  exit 1
fi

VERSION="$1"
ARCH="$2"

IMAGE_REPO="${STANDARD_IMAGE_REPO:-ghcr.io/superplanehq/superplane}"
DEV_BASE_IMAGE_REPO="${DEV_BASE_IMAGE_REPO:-ghcr.io/superplanehq/superplane-dev-base}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# shellcheck source=../lib/image-build-prerequisites.sh
source "${REPO_ROOT}/release/lib/image-build-prerequisites.sh"

cd "${REPO_ROOT}"

require_release_image_build_prerequisites

if generated_protobuf_missing; then
  echo "Generating protobuf files"
  make dev.up
  make pb.gen.models
  make pb.gen.gateway
fi

echo "Building SuperPlane image (${IMAGE_REPO})"

docker buildx build \
  --platform "linux/${ARCH}" \
  --progress=plain \
  --provenance=false \
  --push \
  --target runner \
  --cache-from "${DEV_BASE_IMAGE_REPO}:app-latest-${ARCH}" \
  -t "${IMAGE_REPO}:${VERSION}-${ARCH}" \
  -f Dockerfile \
  .
