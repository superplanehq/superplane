#!/usr/bin/env bash

set -euo pipefail

if [ "$#" -ne 5 ]; then
  echo "Usage: release/superplane/verify-image.sh <image> <version> <source-sha> <amd64-source> <arm64-source>" >&2
  exit 1
fi

IMAGE="$1"
VERSION="$2"
SOURCE_SHA="$3"
AMD64_SOURCE="$4"
ARM64_SOURCE="$5"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SKOPEO=(bash "${SCRIPT_DIR}/run-skopeo.sh")

final_manifest="$("${SKOPEO[@]}" inspect --raw "docker://${IMAGE}")"
image_manifests="$(jq '[.manifests[] | select(.platform.os == "linux" and (.platform.architecture == "amd64" or .platform.architecture == "arm64"))]' <<<"${final_manifest}")"
if [ "$(jq 'length' <<<"${image_manifests}")" -ne 2 ]; then
  echo "Error: ${IMAGE} must contain exactly linux/amd64 and linux/arm64 images" >&2
  exit 1
fi
for architecture in amd64 arm64; do
  if [ "$(jq --arg architecture "${architecture}" '[.[] | select(.platform.architecture == $architecture)] | length' <<<"${image_manifests}")" -ne 1 ]; then
    echo "Error: ${IMAGE} does not contain exactly one linux/${architecture} image" >&2
    exit 1
  fi
  labels="$("${SKOPEO[@]}" inspect \
    --override-os linux \
    --override-arch "${architecture}" \
    --format '{{json .Labels}}' \
    "docker://${IMAGE}")"
  if ! jq -e \
    --arg version "${VERSION}" \
    '."org.opencontainers.image.source" == "https://github.com/superplanehq/superplane" and
     ."org.opencontainers.image.version" == $version' \
    <<<"${labels}" >/dev/null; then
    echo "Error: ${IMAGE} linux/${architecture} labels do not match ${VERSION}" >&2
    printf '%s\n' "${labels}" >&2
    exit 1
  fi
  revision="$(jq -r '."org.opencontainers.image.revision"' <<<"${labels}")"
  if [ "${revision}" != "${SOURCE_SHA}" ]; then
    echo "* Keeping existing linux/${architecture} revision ${revision}" >&2
  fi
done

if [ "$(jq '[.manifests[] | select(.annotations."vnd.docker.reference.type" == "attestation-manifest")] | length' <<<"${final_manifest}")" -lt 2 ]; then
  echo "Error: ${IMAGE} does not include per-platform provenance" >&2
  exit 1
fi

for source in "${AMD64_SOURCE}" "${ARM64_SOURCE}"; do
  source_manifest="$("${SKOPEO[@]}" inspect --raw "docker://${source}")"
  while IFS= read -r digest; do
    if ! jq -e --arg digest "${digest}" 'any(.manifests[]; .digest == $digest)' <<<"${final_manifest}" >/dev/null; then
      echo "Error: ${IMAGE} does not contain prepared descriptor ${digest} from ${source}" >&2
      exit 1
    fi
  done < <(jq -r '.manifests[].digest' <<<"${source_manifest}")
done

echo "SuperPlane image ${IMAGE} verified." >&2
