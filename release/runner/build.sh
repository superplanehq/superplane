#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

if [ "${1-}" = "" ]; then
  echo "Usage: release/runner/build.sh <release-id>"
  echo ""
  echo "Examples:"
  echo "  release/runner/build.sh v0.0.1"
  echo "  release/runner/build.sh sha:<40-character-git-sha>"
  exit 1
fi

VERSION="$1"
semver_pattern='^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$'
git_sha_pattern='^sha:([0-9a-f]{40})$'
if [[ ! "${VERSION}" =~ ${semver_pattern} ]] &&
  [[ ! "${VERSION}" =~ ${git_sha_pattern} ]]; then
  echo "Error: version must be v<semantic-version> or sha:<40-character-lowercase-git-sha>" >&2
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
if [[ "${VERSION}" =~ ${git_sha_pattern} ]]; then
  expected_sha="${BASH_REMATCH[1]}"
  actual_sha="$(git -C "${REPO_ROOT}" rev-parse HEAD)"
  if [ "${actual_sha}" != "${expected_sha}" ]; then
    echo "Error: checked-out commit ${actual_sha} does not match ${expected_sha}" >&2
    exit 1
  fi
fi
OUTPUT_DIR="${RUNNER_RELEASE_OUTPUT_DIR:-${REPO_ROOT}/build/runner/${VERSION}}"
STAGING_DIR="${OUTPUT_DIR}/.staging"
ARCHITECTURES=(amd64 arm64)

# shellcheck source=../lib/image-build-prerequisites.sh
source "${REPO_ROOT}/release/lib/image-build-prerequisites.sh"

cd "${REPO_ROOT}"
require_docker_buildx

rm -rf "${OUTPUT_DIR}"
mkdir -p "${STAGING_DIR}"
trap 'rm -rf "${STAGING_DIR}"' EXIT

for architecture in "${ARCHITECTURES[@]}"; do
  archive="runner-linux-${architecture}.tar.gz"
  architecture_staging_dir="${STAGING_DIR}/${architecture}"

  echo "* Building runner ${VERSION} for linux/${architecture}"
  mkdir -p "${architecture_staging_dir}"
  docker buildx build \
    --platform "linux/${architecture}" \
    --progress=plain \
    --provenance=false \
    --build-arg "RUNNER_VERSION=${VERSION}" \
    --target artifact \
    --output "type=local,dest=${architecture_staging_dir}" \
    -f release/runner/Dockerfile \
    .

  cp "${SCRIPT_DIR}/install.sh" "${architecture_staging_dir}/install.sh"
  chmod 0755 \
    "${architecture_staging_dir}/install.sh" \
    "${architecture_staging_dir}/runner"

  echo "* Creating ${archive}"
  tar -C "${architecture_staging_dir}" -czf "${OUTPUT_DIR}/${archive}" \
    install.sh \
    runner
done

echo "* Creating checksums.txt"
(
  cd "${OUTPUT_DIR}"
  archives=(runner-linux-amd64.tar.gz runner-linux-arm64.tar.gz)
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "${archives[@]}" > checksums.txt
  else
    shasum -a 256 "${archives[@]}" > checksums.txt
  fi
)

rm -rf "${STAGING_DIR}"
trap - EXIT

echo ""
echo "Runner release ${VERSION} is ready in ${OUTPUT_DIR}:"
echo "  runner-linux-amd64.tar.gz"
echo "  runner-linux-arm64.tar.gz"
echo "  checksums.txt"
