#!/usr/bin/env bash

set -euo pipefail

if [ "$#" -ne 3 ]; then
  echo "Usage: release/superplane/publish-architecture.sh <version> <architecture> <oci-archive>" >&2
  exit 1
fi

VERSION="$1"
ARCHITECTURE="$2"
OCI_ARCHIVE="$3"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
IMAGE_REPOSITORY="${SUPERPLANE_IMAGE_REPO:-ghcr.io/superplanehq/superplane}"
IMAGE="${IMAGE_REPOSITORY}:${VERSION}-${ARCHITECTURE}"
SKOPEO=(bash "${SCRIPT_DIR}/run-skopeo.sh")

node "${SCRIPT_DIR}/../component-release.js" validate-version "${VERSION}" >/dev/null
case "${ARCHITECTURE}" in
  amd64 | arm64) ;;
  *)
    echo "Error: architecture must be amd64 or arm64" >&2
    exit 1
    ;;
esac
test -f "${OCI_ARCHIVE}"

inspect_error="$(mktemp)"
raw_manifest="$(mktemp)"
trap 'rm -f "${inspect_error}" "${raw_manifest}"' EXIT

# Hash the index bytes. skopeo inspect without --raw selects the agent
# architecture, so an arm64 index fails on an amd64 machine.
raw_digest() {
  if ! "${SKOPEO[@]}" inspect --raw "$1" >"${raw_manifest}" 2>"${inspect_error}"; then
    return 1
  fi
  sha256sum "${raw_manifest}" | awk '{print "sha256:" $1}'
}

if ! prepared_digest="$(raw_digest "oci-archive:${OCI_ARCHIVE}")"; then
  echo "Error: could not inspect prepared archive ${OCI_ARCHIVE}" >&2
  cat "${inspect_error}" >&2
  exit 1
fi
reused=0

if remote_digest="$(raw_digest "docker://${IMAGE}")"; then
  echo "* Reusing existing image ${IMAGE}@${remote_digest}"
  reused=1
elif grep -Eqi 'manifest unknown|name unknown|not found' "${inspect_error}"; then
  echo "* Publishing new image ${IMAGE}"
  "${SKOPEO[@]}" copy --all "oci-archive:${OCI_ARCHIVE}" "docker://${IMAGE}"
else
  echo "Error: could not determine whether ${IMAGE} exists" >&2
  cat "${inspect_error}" >&2
  exit 1
fi

if ! published_digest="$(raw_digest "docker://${IMAGE}")"; then
  echo "Error: could not inspect published image ${IMAGE}" >&2
  cat "${inspect_error}" >&2
  exit 1
fi
if [ "${reused}" -eq 0 ] && [ "${published_digest}" != "${prepared_digest}" ]; then
  echo "Error: published digest ${published_digest} does not match ${prepared_digest}" >&2
  exit 1
fi
printf '%s\n' "${published_digest}"
