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
(
  cd "${OUTPUT_DIR}"
  sha256sum runner-linux-amd64.tar.gz runner-linux-arm64.tar.gz >checksums.txt
)

export FAKE_S3_ROOT="${TEST_ROOT}/s3"
export FAKE_S3_LOG="${TEST_ROOT}/uploads.log"
export PATH="${TEST_ROOT}/bin:${PATH}"
export RUNNER_RELEASE_OUTPUT_DIR="${OUTPUT_DIR}"

if semantic_error="$(bash "${SCRIPT_DIR}/upload.sh" v1.2.3 s3://release-bucket 2>&1)"; then
  printf 'expected semantic version upload to fail\n' >&2
  exit 1
fi
if [[ "${semantic_error}" != *"S3 uploads only support SHA development releases"* ]]; then
  printf 'semantic version failed for the wrong reason:\n%s\n' "${semantic_error}" >&2
  exit 1
fi

# SHA development releases keep their existing completion-marker behavior.
sha_version="sha:$(printf 'a%.0s' {1..40})"
bash "${SCRIPT_DIR}/upload.sh" "${sha_version}" s3://release-bucket

expected_order="$(printf '%s\n' \
  "runner/${sha_version}/runner-linux-amd64.tar.gz" \
  "runner/${sha_version}/runner-linux-arm64.tar.gz" \
  "runner/${sha_version}/checksums.txt")"
if [ "$(cat "${FAKE_S3_LOG}")" != "${expected_order}" ]; then
  printf 'unexpected upload order:\n%s\n' "$(cat "${FAKE_S3_LOG}")" >&2
  exit 1
fi

if bash "${SCRIPT_DIR}/upload.sh" "${sha_version}" s3://release-bucket; then
  printf 'expected completed SHA release retry to fail\n' >&2
  exit 1
fi

printf 'Runner upload tests passed.\n'
