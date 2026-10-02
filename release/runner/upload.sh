#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

if [ "${1-}" = "" ] || [ "${2-}" = "" ]; then
  echo "Usage: release/runner/upload.sh <version> <s3-bucket-uri>"
  echo ""
  echo "Example:"
  echo "  release/runner/upload.sh v0.0.1 s3://superplane-releases"
  exit 1
fi

VERSION="$1"
S3_BUCKET_URI="${2%/}"

if [[ ! "${VERSION}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([+-][0-9A-Za-z.-]+)?$ ]]; then
  echo "Error: version must be a v-prefixed semantic version, for example v0.0.1" >&2
  exit 1
fi

if [[ ! "${S3_BUCKET_URI}" =~ ^s3://[^/]+$ ]]; then
  echo "Error: S3 bucket URI must have the form s3://bucket-name" >&2
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
OUTPUT_DIR="${RUNNER_RELEASE_OUTPUT_DIR:-${REPO_ROOT}/build/runner/${VERSION}}"
DESTINATION="${S3_BUCKET_URI}/runner/${VERSION}"
ARCHIVES=(runner-linux-amd64.tar.gz runner-linux-arm64.tar.gz)

# shellcheck source=../lib/image-build-prerequisites.sh
source "${REPO_ROOT}/release/lib/image-build-prerequisites.sh"

require_command aws "Install and authenticate the AWS CLI."

for artifact in "${ARCHIVES[@]}" checksums.txt; do
  if [ ! -f "${OUTPUT_DIR}/${artifact}" ]; then
    echo "Error: ${OUTPUT_DIR}/${artifact} does not exist." >&2
    echo "Run release/runner/release.sh ${VERSION} first." >&2
    exit 1
  fi
done

echo "* Verifying runner release checksums"
(
  cd "${OUTPUT_DIR}"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum --check checksums.txt
  else
    shasum -a 256 --check checksums.txt
  fi
)

for archive in "${ARCHIVES[@]}"; do
  echo "* Uploading ${archive}"
  aws s3 cp \
    "${OUTPUT_DIR}/${archive}" \
    "${DESTINATION}/${archive}" \
    --content-type "application/gzip" \
    --cache-control "public,max-age=31536000,immutable" \
    --only-show-errors
done

echo "* Uploading checksums.txt"
aws s3 cp \
  "${OUTPUT_DIR}/checksums.txt" \
  "${DESTINATION}/checksums.txt" \
  --content-type "text/plain" \
  --cache-control "public,max-age=31536000,immutable" \
  --only-show-errors

echo ""
echo "Runner release ${VERSION} uploaded to ${DESTINATION}/"
