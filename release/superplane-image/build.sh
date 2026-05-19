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

push_flag=(--push)
output_ref="${IMAGE_REPO}:${VERSION}-${ARCH}"
if [[ "${PUSH:-1}" == "0" ]]; then
  push_flag=(--load)
  output_ref="${LOCAL_TAG:-superplane:runner-verify}"
  echo "PUSH=0: build only, loading locally as ${output_ref}"
fi

docker buildx build \
  --platform "linux/${ARCH}" \
  --progress=plain \
  --provenance=false \
  "${push_flag[@]}" \
  --cache-from ghcr.io/superplanehq/superplane-dev-base:app-latest \
  -t "${output_ref}" \
  -f Dockerfile \
  .
