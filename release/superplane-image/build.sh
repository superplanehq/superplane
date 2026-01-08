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

echo "Building SuperPlane image"

docker build \
  --push \
  -t "ghcr.io/superplanehq/superplane:${VERSION}-${ARCH}" \
  -f Dockerfile .
