#!/usr/bin/env bash

set -euo pipefail

if [ "$#" -ne 3 ]; then
  echo "Usage: release/superplane/build-assets.sh <version> <source-sha> <output-directory>" >&2
  exit 1
fi

VERSION="$1"
SOURCE_SHA="$2"
OUTPUT_DIR="$3"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
CHART_VERSION="${VERSION#v}"

node "${REPO_ROOT}/release/component-release.js" validate-version "${VERSION}" >/dev/null
if [[ ! "${SOURCE_SHA}" =~ ^[0-9a-f]{40}$ ]]; then
  echo "Error: source SHA must be a full lowercase Git SHA" >&2
  exit 1
fi

rm -rf "${OUTPUT_DIR}"
mkdir -p "${OUTPUT_DIR}"

cd "${REPO_ROOT}"
if [ "${COMPONENT_RELEASES_ALREADY_RESOLVED:-}" = "yes" ]; then
  echo "* Using component release defaults resolved for this release"
else
  echo "* Resolving component release defaults"
  node release/superplane-helm-chart/update-component-release-defaults.js
fi
export COMPONENT_RELEASES_ALREADY_RESOLVED=yes
bash release/superplane-single-host-tarball/build.sh "${VERSION}"
single_host_dir="build/superplane-single-host-tarball-${VERSION}"
cp "${single_host_dir}/superplane-single-host.tar.gz" "${OUTPUT_DIR}/"
cp "${single_host_dir}/superplane-sbom.json" "${OUTPUT_DIR}/"

bash release/superplane-helm-chart/build.sh "${VERSION}" "${OUTPUT_DIR}"
chart="superplane-${CHART_VERSION}.tgz"
test -f "${OUTPUT_DIR}/${chart}"

echo "SuperPlane release assets ${VERSION} are ready in ${OUTPUT_DIR}."
