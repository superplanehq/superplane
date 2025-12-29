#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

if [ "${1-}" = "" ]; then
  echo "Usage: release/demo/build.sh <version>"
  exit 1
fi

VERSION="$1"

echo "Building SuperPlane demo image"

docker buildx build \
  --push \
  --platform linux/amd64,linux/arm64 \
  -t "ghcr.io/superplanehq/superplane:${VERSION}" \
  -t "ghcr.io/superplanehq/superplane:stable" \
  -t "ghcr.io/superplanehq/superplane:beta" \
  -f Dockerfile .
