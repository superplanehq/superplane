#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_ROOT="$(mktemp -d)"
trap 'rm -rf "${TEST_ROOT}"' EXIT

OUTPUT_DIR="${TEST_ROOT}/output"
mkdir -p "${OUTPUT_DIR}" "${TEST_ROOT}/bin" "${TEST_ROOT}/s3"
ln -s "${SCRIPT_DIR}/testdata/aws" "${TEST_ROOT}/bin/aws"
printf 'amd64 archive\n' >"${OUTPUT_DIR}/runner-linux-amd64.tar.gz"
printf 'arm64 archive\n' >"${OUTPUT_DIR}/runner-linux-arm64.tar.gz"
printf '{"spdxVersion":"SPDX-2.3"}\n' >"${OUTPUT_DIR}/runner.spdx.json"
(
  cd "${OUTPUT_DIR}"
  sha256sum runner-linux-amd64.tar.gz runner-linux-arm64.tar.gz runner.spdx.json >checksums.txt
)

export FAKE_S3_ROOT="${TEST_ROOT}/s3"
export FAKE_S3_LOG="${TEST_ROOT}/uploads.log"
export PATH="${TEST_ROOT}/bin:${PATH}"
export RUNNER_RELEASE_OUTPUT_DIR="${OUTPUT_DIR}"

bash "${SCRIPT_DIR}/upload.sh" v1.2.3-rc.1 s3://release-bucket

expected_order="$(printf '%s\n' \
  runner/v1.2.3-rc.1/runner-linux-amd64.tar.gz \
  runner/v1.2.3-rc.1/runner-linux-arm64.tar.gz \
  runner/v1.2.3-rc.1/runner.spdx.json \
  runner/v1.2.3-rc.1/checksums.txt)"
if [ "$(cat "${FAKE_S3_LOG}")" != "${expected_order}" ]; then
  printf 'unexpected upload order:\n%s\n' "$(cat "${FAKE_S3_LOG}")" >&2
  exit 1
fi

# A partial publish retry can reuse every matching immutable object.
bash "${SCRIPT_DIR}/upload.sh" v1.2.3-rc.1 s3://release-bucket

# A version collision must fail instead of replacing content.
printf 'different amd64 archive\n' >"${OUTPUT_DIR}/runner-linux-amd64.tar.gz"
(
  cd "${OUTPUT_DIR}"
  sha256sum runner-linux-amd64.tar.gz runner-linux-arm64.tar.gz runner.spdx.json >checksums.txt
)
if bash "${SCRIPT_DIR}/upload.sh" v1.2.3-rc.1 s3://release-bucket; then
  printf 'expected mismatched immutable object to fail\n' >&2
  exit 1
fi

# SHA development releases keep their existing completion-marker behavior.
rm -rf "${TEST_ROOT}/s3"
mkdir -p "${TEST_ROOT}/s3"
: >"${FAKE_S3_LOG}"
printf 'amd64 archive\n' >"${OUTPUT_DIR}/runner-linux-amd64.tar.gz"
(
  cd "${OUTPUT_DIR}"
  sha256sum runner-linux-amd64.tar.gz runner-linux-arm64.tar.gz >checksums.txt
)
sha_version="sha:$(printf 'a%.0s' {1..40})"
bash "${SCRIPT_DIR}/upload.sh" "${sha_version}" s3://release-bucket
if bash "${SCRIPT_DIR}/upload.sh" "${sha_version}" s3://release-bucket; then
  printf 'expected completed SHA release retry to fail\n' >&2
  exit 1
fi

printf 'Runner upload tests passed.\n'
