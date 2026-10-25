#!/usr/bin/env bash

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEST_DIR="$(mktemp -d)"
trap 'rm -rf "${TEST_DIR}"' EXIT

export ARTIFACT_TEST_STORE="${TEST_DIR}/store"
export PATH="${REPO_ROOT}/release/test-fixtures:${PATH}"

WORK_DIR="${TEST_DIR}/work"
VERSION="v0.1.0-rc.1"
CLI_DIR="${WORK_DIR}/build/cli/${VERSION}"
RUNNER_DIR="${WORK_DIR}/build/runner/${VERSION}"
FLEET_MANAGER_DIR="${WORK_DIR}/build/fleet-manager/${VERSION}"
SUPERPLANE_DIR="${WORK_DIR}/build/superplane/${VERSION}"

mkdir -p "${CLI_DIR}" "${RUNNER_DIR}" "${FLEET_MANAGER_DIR}" "${SUPERPLANE_DIR}"
printf 'cli-checksums\n' >"${CLI_DIR}/checksums.txt"
printf 'runner-checksums\n' >"${RUNNER_DIR}/checksums.txt"
printf 'fleet-manager-checksums\n' >"${FLEET_MANAGER_DIR}/checksums.txt"
printf 'superplane-checksums\n' >"${SUPERPLANE_DIR}/checksums.txt"

cd "${WORK_DIR}"
bash "${REPO_ROOT}/release/component-release-artifact.sh" push cli "${VERSION}" "${CLI_DIR}"
bash "${REPO_ROOT}/release/component-release-artifact.sh" push runner "${VERSION}" "${RUNNER_DIR}"
bash "${REPO_ROOT}/release/component-release-artifact.sh" push fleet-manager "${VERSION}" "${FLEET_MANAGER_DIR}"
bash "${REPO_ROOT}/release/component-release-artifact.sh" push superplane "${VERSION}" "${SUPERPLANE_DIR}"

test -f "${ARTIFACT_TEST_STORE}/cli-${VERSION}.tar.gz"
test -f "${ARTIFACT_TEST_STORE}/runner-${VERSION}.tar.gz"
test -f "${ARTIFACT_TEST_STORE}/fleet-manager-${VERSION}.tar.gz"
test -f "${ARTIFACT_TEST_STORE}/superplane-${VERSION}.tar.gz"

mv "${WORK_DIR}/build" "${WORK_DIR}/prepared"
mkdir -p "${WORK_DIR}/build/cli" "${WORK_DIR}/build/runner" "${WORK_DIR}/build/fleet-manager" "${WORK_DIR}/build/superplane"

bash "${REPO_ROOT}/release/component-release-artifact.sh" pull cli "${VERSION}" "${CLI_DIR}"
bash "${REPO_ROOT}/release/component-release-artifact.sh" pull runner "${VERSION}" "${RUNNER_DIR}"
bash "${REPO_ROOT}/release/component-release-artifact.sh" pull fleet-manager "${VERSION}" "${FLEET_MANAGER_DIR}"
bash "${REPO_ROOT}/release/component-release-artifact.sh" pull superplane "${VERSION}" "${SUPERPLANE_DIR}"

cmp "${WORK_DIR}/prepared/cli/${VERSION}/checksums.txt" "${CLI_DIR}/checksums.txt"
cmp "${WORK_DIR}/prepared/runner/${VERSION}/checksums.txt" "${RUNNER_DIR}/checksums.txt"
cmp "${WORK_DIR}/prepared/fleet-manager/${VERSION}/checksums.txt" "${FLEET_MANAGER_DIR}/checksums.txt"
cmp "${WORK_DIR}/prepared/superplane/${VERSION}/checksums.txt" "${SUPERPLANE_DIR}/checksums.txt"

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

SPLIT_VERSION="v0.1.0-rc.4"
SPLIT_LIMIT=32
SPLIT_WORK_DIR="${TEST_DIR}/split-work"
SPLIT_RELEASE_DIR="${SPLIT_WORK_DIR}/build/runner/${SPLIT_VERSION}"
SPLIT_PAYLOAD="$(printf 'runner-oci-tar-bytes-that-exceed-limit')"
mkdir -p "${SPLIT_RELEASE_DIR}"
printf 'runner-checksums\n' >"${SPLIT_RELEASE_DIR}/checksums.txt"
printf '%s' "${SPLIT_PAYLOAD}" >"${SPLIT_RELEASE_DIR}/runner.oci.tar"
test "$(wc -c <"${SPLIT_RELEASE_DIR}/runner.oci.tar" | tr -d ' ')" -gt "${SPLIT_LIMIT}"

cd "${SPLIT_WORK_DIR}"
COMPONENT_RELEASE_ARTIFACT_MAX_BYTES="${SPLIT_LIMIT}" \
  bash "${REPO_ROOT}/release/component-release-artifact.sh" \
  push runner "${SPLIT_VERSION}" "${SPLIT_RELEASE_DIR}"

SPLIT_BUNDLE="${ARTIFACT_TEST_STORE}/runner-${SPLIT_VERSION}.tar.gz"
tar -tzf "${SPLIT_BUNDLE}" >"${TEST_DIR}/split-bundle-paths"
if grep -F "runner.oci.tar" "${TEST_DIR}/split-bundle-paths"; then
  echo "Error: runner.oci.tar is still inside the release bundle" >&2
  exit 1
fi
grep -F "${SPLIT_VERSION}/checksums.txt" "${TEST_DIR}/split-bundle-paths" >/dev/null
grep -F "${SPLIT_VERSION}/semaphore-artifact-parts.json" "${TEST_DIR}/split-bundle-paths" >/dev/null
node - "${SPLIT_BUNDLE}" "${SPLIT_VERSION}" "${ARTIFACT_TEST_STORE}" "${SPLIT_LIMIT}" <<'JS'
const fs = require("fs");
const { execFileSync } = require("child_process");

const [bundle, version, store, limitRaw] = process.argv.slice(2);
const limit = Number(limitRaw);
const manifest = execFileSync("tar", ["-xOf", bundle, `${version}/semaphore-artifact-parts.json`], {
  encoding: "utf8",
});
const listed = JSON.parse(manifest);
const parts = listed.files.find((file) => file.path === "runner.oci.tar").parts;
if (!Array.isArray(parts) || parts.length < 2) {
  console.error("Error: large release file was not split into multiple artifacts");
  process.exit(1);
}
for (const part of parts) {
  const stored = `${store}/${part}`;
  const size = fs.statSync(stored).size;
  if (size > limit) {
    console.error(`Error: artifact part ${part} is ${size} bytes`);
    process.exit(1);
  }
}
JS

mv "${SPLIT_WORK_DIR}/build" "${SPLIT_WORK_DIR}/prepared"
mkdir -p "${SPLIT_WORK_DIR}/build/runner"
COMPONENT_RELEASE_ARTIFACT_MAX_BYTES="${SPLIT_LIMIT}" \
  bash "${REPO_ROOT}/release/component-release-artifact.sh" \
  pull runner "${SPLIT_VERSION}" "${SPLIT_RELEASE_DIR}"
cmp "${SPLIT_WORK_DIR}/prepared/runner/${SPLIT_VERSION}/checksums.txt" \
  "${SPLIT_RELEASE_DIR}/checksums.txt"
cmp "${SPLIT_WORK_DIR}/prepared/runner/${SPLIT_VERSION}/runner.oci.tar" \
  "${SPLIT_RELEASE_DIR}/runner.oci.tar"

SUM_VERSION="v0.1.0-rc.7"
SUM_PART_LIMIT=40
SUM_BUNDLE_LIMIT=50
SUM_WORK_DIR="${TEST_DIR}/sum-work"
SUM_RELEASE_DIR="${SUM_WORK_DIR}/build/runner/${SUM_VERSION}"
SUM_CHECKSUMS="$(printf '%030d' 1)"
SUM_ARCHIVE="$(printf '%030d' 2)"
mkdir -p "${SUM_RELEASE_DIR}"
printf '%s' "${SUM_CHECKSUMS}" >"${SUM_RELEASE_DIR}/checksums.txt"
printf '%s' "${SUM_ARCHIVE}" >"${SUM_RELEASE_DIR}/runner.oci.tar"
test "$(wc -c <"${SUM_RELEASE_DIR}/checksums.txt" | tr -d ' ')" -le "${SUM_PART_LIMIT}"
test "$(wc -c <"${SUM_RELEASE_DIR}/runner.oci.tar" | tr -d ' ')" -le "${SUM_PART_LIMIT}"

cd "${SUM_WORK_DIR}"
COMPONENT_RELEASE_ARTIFACT_MAX_BYTES="${SUM_PART_LIMIT}" \
  COMPONENT_RELEASE_ARTIFACT_BUNDLE_LIMIT_BYTES="${SUM_BUNDLE_LIMIT}" \
  bash "${REPO_ROOT}/release/component-release-artifact.sh" \
  push runner "${SUM_VERSION}" "${SUM_RELEASE_DIR}"
tar -tzf "${ARTIFACT_TEST_STORE}/runner-${SUM_VERSION}.tar.gz" >"${TEST_DIR}/sum-bundle-paths"
CHECKSUMS_IN_BUNDLE=0
ARCHIVE_IN_BUNDLE=0
if grep -F "${SUM_VERSION}/checksums.txt" "${TEST_DIR}/sum-bundle-paths" >/dev/null; then
  CHECKSUMS_IN_BUNDLE=1
fi
if grep -F "${SUM_VERSION}/runner.oci.tar" "${TEST_DIR}/sum-bundle-paths" >/dev/null; then
  ARCHIVE_IN_BUNDLE=1
fi
if [ "$((CHECKSUMS_IN_BUNDLE + ARCHIVE_IN_BUNDLE))" -ne 1 ]; then
  echo "Error: files under the part limit still overflow one Semaphore artifact" >&2
  exit 1
fi
mv "${SUM_WORK_DIR}/build" "${SUM_WORK_DIR}/prepared"
mkdir -p "${SUM_WORK_DIR}/build/runner"
COMPONENT_RELEASE_ARTIFACT_MAX_BYTES="${SUM_PART_LIMIT}" \
  COMPONENT_RELEASE_ARTIFACT_BUNDLE_LIMIT_BYTES="${SUM_BUNDLE_LIMIT}" \
  bash "${REPO_ROOT}/release/component-release-artifact.sh" \
  pull runner "${SUM_VERSION}" "${SUM_RELEASE_DIR}"
cmp "${SUM_WORK_DIR}/prepared/runner/${SUM_VERSION}/checksums.txt" \
  "${SUM_RELEASE_DIR}/checksums.txt"
cmp "${SUM_WORK_DIR}/prepared/runner/${SUM_VERSION}/runner.oci.tar" \
  "${SUM_RELEASE_DIR}/runner.oci.tar"

TAMPER_VERSION="v0.1.0-rc.5"
TAMPER_WORK_DIR="${TEST_DIR}/tamper-work"
TAMPER_RELEASE_DIR="${TAMPER_WORK_DIR}/build/runner/${TAMPER_VERSION}"
mkdir -p "${TAMPER_RELEASE_DIR}"
printf 'runner-checksums\n' >"${TAMPER_RELEASE_DIR}/checksums.txt"
printf '%s' "${SPLIT_PAYLOAD}" >"${TAMPER_RELEASE_DIR}/runner.oci.tar"
cd "${TAMPER_WORK_DIR}"
COMPONENT_RELEASE_ARTIFACT_MAX_BYTES="${SPLIT_LIMIT}" \
  bash "${REPO_ROOT}/release/component-release-artifact.sh" \
  push runner "${TAMPER_VERSION}" "${TAMPER_RELEASE_DIR}"
PART_FILE="${ARTIFACT_TEST_STORE}/runner-${TAMPER_VERSION}--runner.oci.tar.part000"
test -f "${PART_FILE}"
node -e 'const fs=require("fs"); const fd=fs.openSync(process.argv[1], "r+"); fs.writeSync(fd, Buffer.from("x"), 0, 1, 0); fs.closeSync(fd);' "${PART_FILE}"
rm -rf "${TAMPER_WORK_DIR}/build"
mkdir -p "${TAMPER_WORK_DIR}/build/runner"
if COMPONENT_RELEASE_ARTIFACT_MAX_BYTES="${SPLIT_LIMIT}" \
  bash "${REPO_ROOT}/release/component-release-artifact.sh" \
  pull runner "${TAMPER_VERSION}" "${TAMPER_RELEASE_DIR}"; then
  echo "Error: altered artifact part was accepted" >&2
  exit 1
fi
test ! -e "${TAMPER_RELEASE_DIR}"

TRAVERSAL_VERSION="v0.1.0-rc.6"
TRAVERSAL_WORK_DIR="${TEST_DIR}/traversal-work"
TRAVERSAL_RELEASE_DIR="${TRAVERSAL_WORK_DIR}/build/runner/${TRAVERSAL_VERSION}"
TRAVERSAL_PAYLOAD="${TEST_DIR}/traversal-payload/${TRAVERSAL_VERSION}"
mkdir -p "${TRAVERSAL_WORK_DIR}/build/runner" "${TRAVERSAL_PAYLOAD}"
printf 'runner-checksums\n' >"${TRAVERSAL_PAYLOAD}/checksums.txt"
cat >"${TRAVERSAL_PAYLOAD}/semaphore-artifact-parts.json" <<'EOF'
{"files":[{"path":"../escaped","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","parts":["runner-v0.1.0-rc.6--../escaped.part000"]}]}
EOF
tar -C "${TEST_DIR}/traversal-payload" -czf \
  "${ARTIFACT_TEST_STORE}/runner-${TRAVERSAL_VERSION}.tar.gz" \
  "${TRAVERSAL_VERSION}"
cd "${TRAVERSAL_WORK_DIR}"
if bash "${REPO_ROOT}/release/component-release-artifact.sh" \
  pull runner "${TRAVERSAL_VERSION}" "${TRAVERSAL_RELEASE_DIR}"; then
  echo "Error: artifact parts path traversal was accepted" >&2
  exit 1
fi
test ! -e "${TRAVERSAL_RELEASE_DIR}"
test ! -e "${TRAVERSAL_WORK_DIR}/build/runner/escaped"
test ! -e "${TRAVERSAL_WORK_DIR}/build/escaped"

echo "Component release artifact tests passed."
