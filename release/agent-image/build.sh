#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

if [ "${1-}" = "" ]; then
  echo "Usage: release/agent-image/build.sh <version>"
  exit 1
fi

if [ "${2-}" = "" ]; then
  echo "Usage: release/agent-image/build.sh <version> <arch>"
  exit 1
fi

VERSION="$1"
ARCH="$2"

IMAGE_REPO="${AGENT_IMAGE_REPO:-ghcr.io/superplanehq/superplane-agent}"

echo "Building SuperPlane Agent image (${IMAGE_REPO})"

docker buildx build \
  --platform "linux/${ARCH}" \
  --progress=plain \
  --provenance=false \
  --push \
  --cache-from ghcr.io/superplanehq/superplane-dev-base:agent-latest \
  -t "${IMAGE_REPO}:${VERSION}-${ARCH}" \
  -f agent/Dockerfile \
  agent
