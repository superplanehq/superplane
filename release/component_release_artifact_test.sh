#!/usr/bin/env bash

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEST_DIR="$(mktemp -d)"
trap 'rm -rf "${TEST_DIR}"' EXIT

export ARTIFACT_TEST_STORE="${TEST_DIR}/store"
export PATH="${REPO_ROOT}/release/test-fixtures:${PATH}"

WORK_DIR="${TEST_DIR}/work"
VERSION="v0.1.0-rc.1"
RUNNER_DIR="${WORK_DIR}/build/runner/${VERSION}"
FLEET_MANAGER_DIR="${WORK_DIR}/build/fleet-manager/${VERSION}"

mkdir -p "${RUNNER_DIR}" "${FLEET_MANAGER_DIR}"
printf 'runner-checksums\n' >"${RUNNER_DIR}/checksums.txt"
printf 'fleet-manager-checksums\n' >"${FLEET_MANAGER_DIR}/checksums.txt"

cd "${WORK_DIR}"
bash "${REPO_ROOT}/release/component-release-artifact.sh" push runner "${VERSION}" "${RUNNER_DIR}"
bash "${REPO_ROOT}/release/component-release-artifact.sh" push fleet-manager "${VERSION}" "${FLEET_MANAGER_DIR}"

test -f "${ARTIFACT_TEST_STORE}/runner-${VERSION}.tar.gz"
test -f "${ARTIFACT_TEST_STORE}/fleet-manager-${VERSION}.tar.gz"

mv "${WORK_DIR}/build" "${WORK_DIR}/prepared"
mkdir -p "${WORK_DIR}/build/runner" "${WORK_DIR}/build/fleet-manager"

bash "${REPO_ROOT}/release/component-release-artifact.sh" pull runner "${VERSION}" "${RUNNER_DIR}"
bash "${REPO_ROOT}/release/component-release-artifact.sh" pull fleet-manager "${VERSION}" "${FLEET_MANAGER_DIR}"

cmp "${WORK_DIR}/prepared/runner/${VERSION}/checksums.txt" "${RUNNER_DIR}/checksums.txt"
cmp "${WORK_DIR}/prepared/fleet-manager/${VERSION}/checksums.txt" "${FLEET_MANAGER_DIR}/checksums.txt"

MALICIOUS_WORK_DIR="${TEST_DIR}/malicious-work"
MALICIOUS_VERSION="v0.1.0-rc.2"
MALICIOUS_RELEASE_DIR="${MALICIOUS_WORK_DIR}/build/runner/${MALICIOUS_VERSION}"
MALICIOUS_PAYLOAD_DIR="${TEST_DIR}/malicious-payload/${MALICIOUS_VERSION}"
mkdir -p "${MALICIOUS_WORK_DIR}/build/runner" "${MALICIOUS_PAYLOAD_DIR}"
ln -s /tmp "${MALICIOUS_PAYLOAD_DIR}/release-manifest.json"
tar -C "${TEST_DIR}/malicious-payload" -czf \
  "${ARTIFACT_TEST_STORE}/runner-${MALICIOUS_VERSION}.tar.gz" \
  "${MALICIOUS_VERSION}"

cd "${MALICIOUS_WORK_DIR}"
if bash "${REPO_ROOT}/release/component-release-artifact.sh" \
  pull runner "${MALICIOUS_VERSION}" "${MALICIOUS_RELEASE_DIR}"; then
  echo "Error: unsafe release bundle was accepted" >&2
  exit 1
fi
test ! -e "${MALICIOUS_RELEASE_DIR}"

CORRUPT_WORK_DIR="${TEST_DIR}/corrupt-work"
CORRUPT_VERSION="v0.1.0-rc.3"
CORRUPT_RELEASE_DIR="${CORRUPT_WORK_DIR}/build/runner/${CORRUPT_VERSION}"
mkdir -p "${CORRUPT_WORK_DIR}/build/runner"
printf 'not a gzip archive\n' >\
  "${ARTIFACT_TEST_STORE}/runner-${CORRUPT_VERSION}.tar.gz"

cd "${CORRUPT_WORK_DIR}"
if bash "${REPO_ROOT}/release/component-release-artifact.sh" \
  pull runner "${CORRUPT_VERSION}" "${CORRUPT_RELEASE_DIR}"; then
  echo "Error: corrupt release bundle was accepted" >&2
  exit 1
fi
test ! -e "${CORRUPT_RELEASE_DIR}"

echo "Component release artifact tests passed."
