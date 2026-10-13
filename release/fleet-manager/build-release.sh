#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

if [ "$#" -ne 2 ]; then
  echo "Usage: release/fleet-manager/build-release.sh <version> <git-sha>" >&2
  exit 1
fi

VERSION="$1"
GIT_SHA="$2"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
OUTPUT_DIR="${FLEET_MANAGER_RELEASE_OUTPUT_DIR:-${REPO_ROOT}/build/fleet-manager/${VERSION}}"
STAGING_DIR="${OUTPUT_DIR}/.staging"
ARCHITECTURES=(amd64 arm64)

node "${REPO_ROOT}/release/component-release.js" validate-version "${VERSION}" >/dev/null
if [[ ! "${GIT_SHA}" =~ ^[0-9a-f]{40}$ ]]; then
  echo "Error: git-sha must be a full lowercase 40-character commit SHA" >&2
  exit 1
fi
if [ "$(git -C "${REPO_ROOT}" rev-parse HEAD)" != "${GIT_SHA}" ]; then
  echo "Error: checked-out commit does not match ${GIT_SHA}" >&2
  exit 1
fi

# shellcheck source=../lib/image-build-prerequisites.sh
source "${REPO_ROOT}/release/lib/image-build-prerequisites.sh"
require_docker_buildx

rm -rf "${OUTPUT_DIR}"
mkdir -p "${STAGING_DIR}/sbom-input"
trap 'rm -rf "${STAGING_DIR}"' EXIT

cd "${REPO_ROOT}"
for architecture in "${ARCHITECTURES[@]}"; do
  architecture_dir="${STAGING_DIR}/${architecture}"
  image="superplane-fleet-manager-release:${VERSION#v}-${architecture}"
  mkdir -p "${architecture_dir}"

  docker buildx build \
    --platform "linux/${architecture}" \
    --progress=plain \
    --provenance=false \
    --build-arg "FLEET_MANAGER_VERSION=${VERSION}" \
    --build-arg "SOURCE_REVISION=${GIT_SHA}" \
    --target runtime \
    --load \
    --tag "${image}" \
    -f release/fleet-manager/Dockerfile \
    .

  container="$(docker create "${image}")"
  docker cp "${container}:/usr/local/bin/fleet-manager" "${architecture_dir}/fleet-manager"
  docker rm "${container}" >/dev/null
  chmod 0755 "${architecture_dir}/fleet-manager"

  reported_version="$(docker run --rm --platform "linux/${architecture}" "${image}" --version)"
  if [ "${reported_version}" != "fleet-manager ${VERSION}" ]; then
    echo "Error: linux/${architecture} Fleet Manager reported ${reported_version}" >&2
    exit 1
  fi

  cp "${architecture_dir}/fleet-manager" \
    "${STAGING_DIR}/sbom-input/fleet-manager-linux-${architecture}"
  tar -C "${architecture_dir}" -czf \
    "${OUTPUT_DIR}/fleet-manager-linux-${architecture}.tar.gz" \
    fleet-manager
done

bash "${REPO_ROOT}/release/generate-component-sbom.sh" \
  fleet-manager \
  "${STAGING_DIR}/sbom-input" \
  "${OUTPUT_DIR}/fleet-manager.spdx.json"

(
  cd "${OUTPUT_DIR}"
  artifacts=(
    fleet-manager-linux-amd64.tar.gz
    fleet-manager-linux-arm64.tar.gz
    fleet-manager.spdx.json
  )
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "${artifacts[@]}" >checksums.txt
  else
    shasum -a 256 "${artifacts[@]}" >checksums.txt
  fi
)

docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --progress=plain \
  --provenance=mode=max \
  --build-arg "FLEET_MANAGER_VERSION=${VERSION}" \
  --build-arg "SOURCE_REVISION=${GIT_SHA}" \
  --target runtime \
  --output "type=oci,dest=${OUTPUT_DIR}/fleet-manager.oci.tar" \
  -f release/fleet-manager/Dockerfile \
  .

bash "${SCRIPT_DIR}/verify-oci-archive.sh" \
  "${OUTPUT_DIR}/fleet-manager.oci.tar" \
  "${VERSION}" \
  "${GIT_SHA}"

rm -rf "${STAGING_DIR}"
trap - EXIT

echo "Fleet Manager release ${VERSION} is ready in ${OUTPUT_DIR}."
