#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

if [ "${1-}" = "" ]; then
  echo "Usage: release/superplane-demo-image/build.sh <version> <arch>"
  exit 1
fi

if [ "${2-}" = "" ]; then
  echo "Usage: release/superplane-demo-image/build.sh <version> <arch>"
  exit 1
fi

VERSION="$1"
ARCH="$2"

IMAGE_REPO="${DEMO_IMAGE_REPO:-ghcr.io/superplanehq/superplane-demo}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

SUPERGIT_VERSION="${SUPERGIT_VERSION:-v0.1.1}"

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

bash "${SCRIPT_DIR}/download-supergit.sh" "${SUPERGIT_VERSION}" "${ARCH}"

echo "Building SuperPlane demo image (${IMAGE_REPO})"

docker buildx build \
  --platform "linux/${ARCH}" \
  --progress=plain \
  --provenance=false \
  --push \
  --target demo \
  -t "${IMAGE_REPO}:${VERSION}-${ARCH}" \
  -f Dockerfile \
  .
