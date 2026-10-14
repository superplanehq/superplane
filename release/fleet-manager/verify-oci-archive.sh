#!/usr/bin/env bash

set -euo pipefail

if [ "$#" -ne 3 ]; then
  echo "Usage: release/fleet-manager/verify-oci-archive.sh <oci-archive> <version> <source-sha>" >&2
  exit 1
fi

OCI_ARCHIVE="$1"
VERSION="$2"
SOURCE_SHA="$3"

read_blob() {
  local digest="$1"
  if [[ ! "${digest}" =~ ^sha256:([0-9a-f]{64})$ ]]; then
    echo "Error: invalid OCI digest ${digest}" >&2
    return 1
  fi
  tar -xOf "${OCI_ARCHIVE}" "blobs/sha256/${BASH_REMATCH[1]}"
}

top_digest="$(tar -xOf "${OCI_ARCHIVE}" index.json | jq -r '.manifests[0].digest')"
index="$(read_blob "${top_digest}")"
if [ "$(jq '[.manifests[] | select(.annotations."vnd.docker.reference.type" == "attestation-manifest")] | length' <<<"${index}")" -lt 2 ]; then
  echo "Error: prepared OCI image does not include per-platform provenance" >&2
  exit 1
fi

for architecture in amd64 arm64; do
  manifest_digest="$(
    jq -r --arg architecture "${architecture}" \
      '.manifests[] | select(.platform.os == "linux" and .platform.architecture == $architecture) | .digest' \
      <<<"${index}"
  )"
  if [ -z "${manifest_digest}" ]; then
    echo "Error: prepared OCI image is missing linux/${architecture}" >&2
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
done

echo "Prepared Fleet Manager OCI archive verified."
