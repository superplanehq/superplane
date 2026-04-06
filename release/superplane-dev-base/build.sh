#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

if [ "${1-}" = "" ]; then
  echo "Usage: release/superplane-dev-base/build.sh <arch>"
  exit 1
fi

ARCH="$1"
IMAGE_REPO="${DEV_BASE_IMAGE_REPO:-ghcr.io/superplanehq/superplane-dev-base}"

echo "Building SuperPlane dev base images (${IMAGE_REPO})"

docker buildx build \
  --platform "linux/${ARCH}" \
  --progress=plain \
  --provenance=false \
  --push \
  --target dev-base \
  --cache-to type=inline \
  -t "${IMAGE_REPO}:app-latest-${ARCH}" \
  -f Dockerfile \
  .

docker buildx build \
  --platform "linux/${ARCH}" \
  --progress=plain \
  --provenance=false \
  --push \
  --target dev \
  --cache-to type=inline \
  -t "${IMAGE_REPO}:agent-latest-${ARCH}" \
  -f agent/Dockerfile \
  agent
