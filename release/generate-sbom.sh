#!/usr/bin/env bash

set -euo pipefail

if [ -z "${1:-}" ]; then
  echo "Usage: release/generate-sbom.sh <version>"
  echo ""
  echo "Example:"
  echo "  release/generate-sbom.sh v1.2.3"
  exit 1
fi

VERSION="$1"
IMAGE_NAME="ghcr.io/superplanehq/superplane:${VERSION}"
BUILD_ROOT="build/superplane-single-host-tarball-${VERSION}"
SBOM_OUTPUT="${BUILD_ROOT}/superplane-sbom.json"

echo "* Generating SBOM for version ${VERSION}"
echo "* Image: ${IMAGE_NAME}"
echo "* Output: ${SBOM_OUTPUT}"

# Check if syft is installed
if ! command -v syft >/dev/null 2>&1; then
  echo "Error: syft is not installed."
  echo "Install it with: curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b /usr/local/bin"
  exit 1
fi

# Create build directory if it doesn't exist
mkdir -p "${BUILD_ROOT}"

# Generate SBOM in SPDX JSON format
syft "${IMAGE_NAME}" -o spdx-json="${SBOM_OUTPUT}"

echo ""
echo "Done."
echo "SBOM: ${SBOM_OUTPUT}"
