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
SYFT_VERSION="v1.54.1"
BUILD_ROOT="build/superplane-single-host-tarball-${VERSION}"
SBOM_OUTPUT="${BUILD_ROOT}/superplane-sbom.json"

# Find the repository root (where go.mod is located)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Syft resolves files to physical paths; exclusions must use the same root.
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd -P)"

echo "* Generating SBOM for version ${VERSION}"
echo "* Source: ${REPO_ROOT}"
echo "* Output: ${SBOM_OUTPUT}"

# Check if syft is installed, install if missing
SYFT_BIN="${REPO_ROOT}/tmp/bin/syft"
if command -v syft >/dev/null 2>&1; then
  SYFT_BIN="syft"
elif [ -x "${SYFT_BIN}" ]; then
  : # Use local installation
else
  echo "* syft not found, installing..."
  mkdir -p "${REPO_ROOT}/tmp/bin"
  curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | \
    sh -s -- -b "${REPO_ROOT}/tmp/bin" "${SYFT_VERSION}"
fi

# Create build directory if it doesn't exist
mkdir -p "${BUILD_ROOT}"

# Scan source manifests, including Go modules and npm lockfiles. Installed
# dependencies, caches, and previous build output are not release inputs.
SYFT_CHECK_FOR_APP_UPDATE=false "${SYFT_BIN}" dir:"${REPO_ROOT}" \
  --exclude './.git/**' \
  --exclude './tmp/**' \
  --exclude './build/**' \
  --exclude '**/node_modules/**' \
  -o spdx-json="${SBOM_OUTPUT}"

echo ""
echo "Done."
echo "SBOM: ${SBOM_OUTPUT}"
