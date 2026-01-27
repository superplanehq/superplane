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
APT_MIRROR="${APT_MIRROR-}"
APT_SECURITY_MIRROR="${APT_SECURITY_MIRROR-}"

echo "Building SuperPlane demo image"

BUILD_ARGS=()
if [ -n "${APT_MIRROR}" ]; then
  BUILD_ARGS+=(--build-arg "APT_MIRROR=${APT_MIRROR}")
fi
if [ -n "${APT_SECURITY_MIRROR}" ]; then
  BUILD_ARGS+=(--build-arg "APT_SECURITY_MIRROR=${APT_SECURITY_MIRROR}")
fi

docker build \
  --push \
  -t "ghcr.io/superplanehq/superplane-demo:${VERSION}-${ARCH}" \
  "${BUILD_ARGS[@]}" \
  -f release/superplane-demo-image/Dockerfile . \
