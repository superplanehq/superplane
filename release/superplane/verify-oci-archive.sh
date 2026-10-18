#!/usr/bin/env bash

set -euo pipefail

if [ "$#" -ne 4 ]; then
  echo "Usage: release/superplane/verify-oci-archive.sh <archive> <version> <source-sha> <architecture>" >&2
  exit 1
fi

ARCHIVE="$1"
VERSION="$2"
SOURCE_SHA="$3"
ARCHITECTURE="$4"

read_blob() {
  local digest="$1"
  if [[ ! "${digest}" =~ ^sha256:([0-9a-f]{64})$ ]]; then
    echo "Error: invalid OCI digest ${digest}" >&2
    return 1
  fi
  tar -xOf "${ARCHIVE}" "blobs/sha256/${BASH_REMATCH[1]}"
}

top_digest="$(tar -xOf "${ARCHIVE}" index.json | jq -r '.manifests[0].digest')"
index="$(read_blob "${top_digest}")"
manifest_digest="$(
  jq -r --arg architecture "${ARCHITECTURE}" \
    '.manifests[] | select(.platform.os == "linux" and .platform.architecture == $architecture) | .digest' \
    <<<"${index}"
)"
if [ -z "${manifest_digest}" ]; then
  echo "Error: prepared OCI image is missing linux/${ARCHITECTURE}" >&2
  exit 1
fi
if [ "$(jq '[.manifests[] | select(.annotations."vnd.docker.reference.type" == "attestation-manifest")] | length' <<<"${index}")" -lt 1 ]; then
  echo "Error: prepared OCI image does not include provenance" >&2
  exit 1
fi

config_digest="$(read_blob "${manifest_digest}" | jq -r '.config.digest')"
labels="$(read_blob "${config_digest}" | jq '.config.Labels')"
jq -e \
  --arg version "${VERSION}" \
  --arg revision "${SOURCE_SHA}" \
  '."org.opencontainers.image.source" == "https://github.com/superplanehq/superplane" and
   ."org.opencontainers.image.version" == $version and
   ."org.opencontainers.image.revision" == $revision' \
  <<<"${labels}" >/dev/null

echo "Prepared linux/${ARCHITECTURE} SuperPlane OCI archive verified."
