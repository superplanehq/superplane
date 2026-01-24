#!/usr/bin/env bash

set -euo pipefail

if [ "${1-}" = "" ]; then
  echo "Usage: release/superplane-single-host-tarball/build.sh <version>"
  echo ""
  echo "Example:"
  echo "  release/superplane-single-host-tarball/build.sh v1.2.3"
  exit 1
fi

VERSION="$1"

BUILD_ROOT="build/superplane-single-host-tarball-${VERSION}"
TARGET_DIR="${BUILD_ROOT}/superplane"
TEMPLATES_DIR="release/superplane-single-host-tarball/templates"

echo "* Building single-host release for version ${VERSION}"
echo "* Target directory: ${TARGET_DIR}"

rm -rf "${BUILD_ROOT}"
mkdir -p "${TARGET_DIR}"

echo "* Injecting docker-compose.yml"
sed "s/__SUPERPLANE_VERSION__/${VERSION}/g" "${TEMPLATES_DIR}/docker-compose.yml" > "${TARGET_DIR}/docker-compose.yml"

echo "* Injecting install.sh"
cp "${TEMPLATES_DIR}/install.sh" "${TARGET_DIR}/install.sh"
chmod +x "${TARGET_DIR}/install.sh"

echo "* Injecting Caddyfile"
cp "${TEMPLATES_DIR}/Caddyfile" "${TARGET_DIR}/Caddyfile"

echo "* Injecting superplane-logo.txt"
cp "${TEMPLATES_DIR}/superplane-logo.txt" "${TARGET_DIR}/superplane-logo.txt"

echo "* Generating SBOM"
release/generate-sbom.sh "${VERSION}"

echo "* Creating superplane-single-host.tar.gz"
(
  cd "${BUILD_ROOT}"
  tar czf superplane-single-host.tar.gz superplane
)

echo ""
echo "Done."
echo "Artifact: ${BUILD_ROOT}/superplane-single-host.tar.gz"
echo "SBOM: ${BUILD_ROOT}/superplane-sbom.json"
