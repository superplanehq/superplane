#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

if [ "${1-}" = "" ] || [ "${2-}" = "" ]; then
  echo "Usage: release/runner/upload.sh <release-id> <s3-bucket-uri>"
  echo ""
  echo "Examples:"
  echo "  release/runner/upload.sh v0.0.1 s3://superplanehq-releases"
  echo "  release/runner/upload.sh sha:<40-character-git-sha> s3://superplanehq-releases"
  exit 1
fi

VERSION="$1"
S3_BUCKET_URI="${2%/}"

semver_pattern='^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$'
git_sha_pattern='^sha:[0-9a-f]{40}$'
if [[ ! "${VERSION}" =~ ${semver_pattern} ]] &&
  [[ ! "${VERSION}" =~ ${git_sha_pattern} ]]; then
  echo "Error: version must be v<semantic-version> or sha:<40-character-lowercase-git-sha>" >&2
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

is_git_sha_release=false
if [[ "${VERSION}" =~ ${git_sha_pattern} ]]; then
  is_git_sha_release=true
  put_object_skeleton="$(aws s3api put-object --generate-cli-skeleton input)"
  if [[ "${put_object_skeleton}" != *'"IfNoneMatch"'* ]]; then
    echo "Error: AWS CLI must support s3api put-object --if-none-match for SHA releases" >&2
    exit 1
  fi
  bucket="${S3_BUCKET_URI#s3://}"
  prefix="runner/${VERSION}/"
  existing_key_count="$(
    aws s3api list-objects-v2 \
      --bucket "${bucket}" \
      --prefix "${prefix}" \
      --max-keys 1 \
      --query KeyCount \
      --output text
  )"
  if [ "${existing_key_count}" != "0" ]; then
    echo "Error: immutable runner release ${DESTINATION}/ already exists" >&2
    exit 1
  fi
fi

upload_artifact() {
  local source_path="$1"
  local object_name="$2"
  local content_type="$3"

  if [ "${is_git_sha_release}" = "true" ]; then
    aws s3api put-object \
      --bucket "${bucket}" \
      --key "${prefix}${object_name}" \
      --body "${source_path}" \
      --content-type "${content_type}" \
      --cache-control "public,max-age=31536000,immutable" \
      --if-none-match "*" \
      >/dev/null
    return
  fi

  aws s3 cp \
    "${source_path}" \
    "${DESTINATION}/${object_name}" \
    --content-type "${content_type}" \
    --cache-control "public,max-age=31536000,immutable" \
    --only-show-errors
}

for artifact in "${ARCHIVES[@]}" checksums.txt; do
  if [ ! -f "${OUTPUT_DIR}/${artifact}" ]; then
    echo "Error: ${OUTPUT_DIR}/${artifact} does not exist." >&2
    echo "Run release/runner/build.sh ${VERSION} first." >&2
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
  upload_artifact "${OUTPUT_DIR}/${archive}" "${archive}" "application/gzip"
done

echo "* Uploading checksums.txt"
upload_artifact "${OUTPUT_DIR}/checksums.txt" "checksums.txt" "text/plain"

echo ""
echo "Runner release ${VERSION} uploaded to ${DESTINATION}/"
