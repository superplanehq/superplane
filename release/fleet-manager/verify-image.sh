#!/usr/bin/env bash

set -euo pipefail

if [ "$#" -ne 4 ]; then
  echo "Usage: release/fleet-manager/verify-image.sh <image> <digest> <version> <source-sha>" >&2
  exit 1
fi

IMAGE="$1"
DIGEST="$2"
VERSION="$3"
SOURCE_SHA="$4"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SKOPEO=(bash "${SCRIPT_DIR}/run-skopeo.sh")

if [[ "${IMAGE}" != *:"${VERSION}" ]]; then
  echo "Error: ${IMAGE} does not use the expected version tag ${VERSION}" >&2
  exit 1
fi

actual_digest="$("${SKOPEO[@]}" inspect --format '{{.Digest}}' "docker://${IMAGE}")"
if [ "${actual_digest}" != "${DIGEST}" ]; then
  echo "Error: ${IMAGE} resolves to ${actual_digest}, not ${DIGEST}" >&2
  exit 1
fi

immutable_image="${IMAGE%:${VERSION}}@${DIGEST}"
raw_manifest="$("${SKOPEO[@]}" inspect --raw "docker://${immutable_image}")"
for architecture in amd64 arm64; do
  if ! jq -e --arg architecture "${architecture}" \
    'any(.manifests[]; .platform.os == "linux" and .platform.architecture == $architecture)' \
    <<<"${raw_manifest}" >/dev/null; then
    echo "Error: ${IMAGE}@${DIGEST} is missing linux/${architecture}" >&2
    exit 1
  fi
  labels="$("${SKOPEO[@]}" inspect --override-os linux --override-arch "${architecture}" \
    --format '{{json .Labels}}' "docker://${immutable_image}")"
  jq -e \
    --arg version "${VERSION}" \
    --arg revision "${SOURCE_SHA}" \
    '."org.opencontainers.image.source" == "https://github.com/superplanehq/superplane" and
     ."org.opencontainers.image.version" == $version and
     ."org.opencontainers.image.revision" == $revision' \
    <<<"${labels}" >/dev/null
done

if ! jq -e \
  'any(.manifests[]; .annotations."vnd.docker.reference.type" == "attestation-manifest")' \
  <<<"${raw_manifest}" >/dev/null; then
  echo "Error: ${IMAGE}@${DIGEST} does not include BuildKit provenance" >&2
  exit 1
fi

echo "Fleet Manager image ${immutable_image} verified."
