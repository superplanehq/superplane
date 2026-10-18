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
git_sha_pattern='^sha:([0-9a-f]{40})$'
is_git_sha_release=false
if [[ "${VERSION}" == v* ]]; then
  if ! node "$(dirname "${BASH_SOURCE[0]}")/../component-release.js" \
    validate-version "${VERSION}" >/dev/null; then
    exit 1
  fi
elif [[ ! "${VERSION}" =~ ${git_sha_pattern} ]]; then
  echo "Error: version must be v<semantic-version> or sha:<40-character-lowercase-git-sha>" >&2
  exit 1
else
  is_git_sha_release=true
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
  release_inputs=(
    go.mod
    go.sum
    cmd/runner
    pkg
    release/runner/Dockerfile
    release/runner/build.sh
    release/runner/install.sh
  )
  input_changes="$(
    git -C "${REPO_ROOT}" status \
      --porcelain \
      --untracked-files=all \
      -- "${release_inputs[@]}"
  )"
  if [ -n "${input_changes}" ]; then
    echo "Error: SHA runner release inputs contain uncommitted changes:" >&2
    echo "${input_changes}" >&2
    exit 1
  fi
fi
OUTPUT_DIR="${RUNNER_RELEASE_OUTPUT_DIR:-${REPO_ROOT}/build/runner/${VERSION}}"
STAGING_DIR="${OUTPUT_DIR}/.staging"
SBOM_INPUT_DIR="${STAGING_DIR}/sbom-input"
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
  image="superplane-runner-release:${VERSION#v}-${architecture}"

  echo "* Building runner ${VERSION} for linux/${architecture}"
  mkdir -p "${architecture_staging_dir}"
  if [ "${is_git_sha_release}" = "true" ]; then
    docker buildx build \
      --platform "linux/${architecture}" \
      --progress=plain \
      --provenance=false \
      --build-arg "RUNNER_VERSION=${VERSION}" \
      --target artifact \
      --output "type=local,dest=${architecture_staging_dir}" \
      -f release/runner/Dockerfile \
      .
  else
    docker buildx build \
      --platform "linux/${architecture}" \
      --progress=plain \
      --provenance=false \
      --build-arg "RUNNER_VERSION=${VERSION}" \
      --target runtime \
      --load \
      --tag "${image}" \
      -f release/runner/Dockerfile \
      .

    container="$(docker create "${image}")"
    docker cp "${container}:/usr/local/bin/runner" "${architecture_staging_dir}/runner"
    docker rm "${container}" >/dev/null

    reported_version="$(
      docker run --rm \
        --platform "linux/${architecture}" \
        "${image}" \
        --version
    )"
    if [ "${reported_version}" != "runner ${VERSION}" ]; then
      echo "Error: linux/${architecture} runner reported ${reported_version}" >&2
      exit 1
    fi

    mkdir -p "${SBOM_INPUT_DIR}"
    cp "${architecture_staging_dir}/runner" "${SBOM_INPUT_DIR}/runner-linux-${architecture}"
  fi

  cp "${SCRIPT_DIR}/install.sh" "${architecture_staging_dir}/install.sh"
  chmod 0755 \
    "${architecture_staging_dir}/install.sh" \
    "${architecture_staging_dir}/runner"

  echo "* Creating ${archive}"
  tar -C "${architecture_staging_dir}" -czf "${OUTPUT_DIR}/${archive}" \
    install.sh \
    runner
done

if [ "${is_git_sha_release}" = "false" ]; then
  echo "* Creating runner.spdx.json"
  bash "${REPO_ROOT}/release/generate-component-sbom.sh" \
    runner \
    "${SBOM_INPUT_DIR}" \
    "${OUTPUT_DIR}/runner.spdx.json"
fi

echo "* Creating checksums.txt"
(
  cd "${OUTPUT_DIR}"
  archives=(runner-linux-amd64.tar.gz runner-linux-arm64.tar.gz)
  if [ "${is_git_sha_release}" = "false" ]; then
    archives+=(runner.spdx.json)
  fi
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
if [ "${is_git_sha_release}" = "false" ]; then
  echo "  runner.spdx.json"
fi
echo "  checksums.txt"
