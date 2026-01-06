#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

if [ "${1-}" = "" ]; then
  echo "Usage: release/superplane-demo-image/build.sh <version>"
  exit 1
fi

VERSION="$1"

echo "Building SuperPlane demo image"

docker buildx build \
  --progress=quiet \
  --push \
  --platform linux/amd64,linux/arm64 \
  -t "ghcr.io/superplanehq/superplane-demo:${VERSION}" \
  -f release/superplane-demo-image/Dockerfile . \
