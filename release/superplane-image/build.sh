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

if [ ! -f "pkg/protos/me/me.pb.go" ] || [ ! -f "pkg/protos/me/me.pb.gw.go" ]; then
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
  --cache-from ghcr.io/superplanehq/superplane-dev-base:app-latest \
  -t "${IMAGE_REPO}:${VERSION}-${ARCH}" \
  -f Dockerfile \
  .
