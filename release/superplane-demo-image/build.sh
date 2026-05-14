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

echo "Building SuperPlane demo image (${IMAGE_REPO})"

docker buildx build \
  --platform "linux/${ARCH}" \
  --progress=plain \
  --provenance=false \
  --push \
  -t "${IMAGE_REPO}:${VERSION}-${ARCH}" \
  -f release/superplane-demo-image/Dockerfile \
  .
