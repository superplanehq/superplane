#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

if [ "${1-}" = "" ]; then
  echo "Usage: release/demo/build.sh <version>"
  exit 1
fi

VERSION="$1"
IMAGE="ghcr.io/superplanehq/superplane-demo:${VERSION}"

echo "Building Superplane demo image: ${IMAGE}"
docker build -f release/demo/Dockerfile -t "${IMAGE}" .