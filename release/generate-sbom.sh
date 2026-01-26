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
BUILD_ROOT="build/superplane-single-host-tarball-${VERSION}"
SBOM_OUTPUT="${BUILD_ROOT}/superplane-sbom.json"

# Find the repository root (where go.mod is located)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

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
  curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b "${REPO_ROOT}/tmp/bin"
fi

# Create build directory if it doesn't exist
mkdir -p "${BUILD_ROOT}"

# Generate SBOM in SPDX JSON format from source files
# This scans go.mod for Go dependencies and web_src/package-lock.json for npm dependencies
"${SYFT_BIN}" dir:"${REPO_ROOT}" -o spdx-json="${SBOM_OUTPUT}"

echo ""
echo "Done."
echo "SBOM: ${SBOM_OUTPUT}"
