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
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
VALUES_FILE="${REPO_ROOT}/release/superplane-helm-chart/helm/values.yaml"

RUNNER_VERSION="$(awk '$1 == "runnerVersion:" { print $2; exit }' "${VALUES_FILE}")"
FLEET_MANAGER_VERSION="$(awk '$1 == "fleetManagerVersion:" { print $2; exit }' "${VALUES_FILE}")"
if [[ -z "${RUNNER_VERSION}" || -z "${FLEET_MANAGER_VERSION}" ]]; then
  echo "Error: runnerVersion and fleetManagerVersion are required in ${VALUES_FILE}" >&2
  exit 1
fi

BUILD_ROOT="${REPO_ROOT}/build/superplane-single-host-tarball-${VERSION}"
TARGET_DIR="${BUILD_ROOT}/superplane"
TEMPLATES_DIR="${SCRIPT_DIR}/templates"

echo "* Building single-host release for version ${VERSION}"
echo "* Runner ${RUNNER_VERSION}, Fleet Manager ${FLEET_MANAGER_VERSION}"
echo "* Target directory: ${TARGET_DIR}"

rm -rf "${BUILD_ROOT}"
mkdir -p "${TARGET_DIR}"

inject() {
  local source="$1"
  local destination="$2"
  sed \
    -e "s/__SUPERPLANE_VERSION__/${VERSION}/g" \
    -e "s/__RUNNER_VERSION__/${RUNNER_VERSION}/g" \
    -e "s/__FLEET_MANAGER_VERSION__/${FLEET_MANAGER_VERSION}/g" \
    "${source}" >"${destination}"
}

echo "* Injecting docker-compose.yml"
inject "${TEMPLATES_DIR}/docker-compose.yml" "${TARGET_DIR}/docker-compose.yml"

for script in install.sh uninstall.sh; do
  echo "* Injecting ${script}"
  inject "${TEMPLATES_DIR}/${script}" "${TARGET_DIR}/${script}"
  chmod +x "${TARGET_DIR}/${script}"
done

echo "* Injecting Caddyfile"
cp "${TEMPLATES_DIR}/Caddyfile" "${TARGET_DIR}/Caddyfile"

echo "* Injecting superplane-logo.txt"
cp "${TEMPLATES_DIR}/superplane-logo.txt" "${TARGET_DIR}/superplane-logo.txt"

echo "* Generating SBOM"
(
  cd "${REPO_ROOT}"
  bash release/generate-sbom.sh "${VERSION}"
)

echo "* Creating superplane-single-host.tar.gz"
(
  cd "${BUILD_ROOT}"
  tar czf superplane-single-host.tar.gz superplane
)

echo ""
echo "Done."
echo "Artifact: ${BUILD_ROOT}/superplane-single-host.tar.gz"
echo "SBOM: ${BUILD_ROOT}/superplane-sbom.json"
