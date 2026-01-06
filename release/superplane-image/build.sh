#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

if [ "${1-}" = "" ]; then
  echo "Usage: release/superplane-image/build.sh <version>"
  exit 1
fi

VERSION="$1"

echo "Building SuperPlane image"

docker buildx build \
  --progress=quiet \
  --push \
  --platform linux/amd64,linux/arm64 \
  -t "ghcr.io/superplanehq/superplane:${VERSION}" \
  -f Dockerfile .
