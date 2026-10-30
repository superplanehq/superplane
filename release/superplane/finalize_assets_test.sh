#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
VALUES_FILE="${REPO_ROOT}/release/superplane-helm-chart/helm/values.yaml"
VERSION="v9.9.8"
RUNNER_VERSION="v8.7.6"
FLEET_MANAGER_VERSION="v6.5.4"
BUILD_ROOT="${REPO_ROOT}/build/superplane-single-host-tarball-${VERSION}"
TMP_DIR="$(mktemp -d)"

cp "${VALUES_FILE}" "${TMP_DIR}/values.yaml.bak"
cleanup() {
  cp "${TMP_DIR}/values.yaml.bak" "${VALUES_FILE}"
  rm -rf "${BUILD_ROOT}" "${TMP_DIR}"
}
trap cleanup EXIT

cat >"${TMP_DIR}/candidate-values.yaml" <<EOF
componentReleases:
  runnerVersion: ${RUNNER_VERSION}
  runnerVersions:
    - ${RUNNER_VERSION}
    - v7.6.5
  fleetManagerVersion: ${FLEET_MANAGER_VERSION}
EOF

bash "${SCRIPT_DIR}/build-finalized-assets.sh" \
  "${TMP_DIR}/candidate-values.yaml" \
  "${VERSION}" \
  "$(printf 'a%.0s' {1..40})" \
  "${TMP_DIR}/assets"

chart_values="${TMP_DIR}/stable-values.yaml"
tar -xOf \
  "${TMP_DIR}/assets/superplane-${VERSION#v}.tgz" \
  superplane-chart/values.yaml \
  >"${chart_values}"
grep -q "runnerVersion: ${RUNNER_VERSION}" "${chart_values}"
grep -q "fleetManagerVersion: ${FLEET_MANAGER_VERSION}" "${chart_values}"

bundle_dir="${TMP_DIR}/bundle"
mkdir -p "${bundle_dir}"
tar -C "${bundle_dir}" -xzf \
  "${TMP_DIR}/assets/superplane-single-host.tar.gz"
grep -q \
  "ghcr.io/superplanehq/fleet-manager:${FLEET_MANAGER_VERSION}-docker" \
  "${bundle_dir}/superplane/docker-compose.yml"
grep -q \
  "RUNNER_VERSION=\"${RUNNER_VERSION}\"" \
  "${bundle_dir}/superplane/install.sh"

echo "SuperPlane finalized asset tests passed."
