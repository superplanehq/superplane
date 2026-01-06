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

echo "Building SuperPlane demo image"

docker build \
  --push \
  -t "ghcr.io/superplanehq/superplane-demo:${VERSION}-${ARCH}" \
  -f release/superplane-demo-image/Dockerfile . \
