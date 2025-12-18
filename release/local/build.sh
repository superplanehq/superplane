#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

if [ "${1-}" = "" ]; then
  echo "Usage: release/local/build.sh <version>"
  exit 1
fi

VERSION="$1"
IMAGE="ghcr.io/superplanehq/superplane-allinone:${VERSION}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

cd "${REPO_ROOT}"

echo "Building Superplane trial image: ${IMAGE}"

docker build \
  -f release/local/Dockerfile \
  -t "${IMAGE}" \
  .