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

echo "Building SuperPlane image (${IMAGE_REPO})"

make gen.setup.backend
make gen.setup.ui

docker buildx build \
  --progress=plain \
  --provenance=false \
  --push \
  --cache-from ghcr.io/superplanehq/superplane-dev-base:app-latest \
  -t "dasdas-dasdasdas" \
  -f Dockerfile \
  .
