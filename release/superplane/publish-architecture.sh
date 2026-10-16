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

prepared_digest="$("${SKOPEO[@]}" inspect --raw "oci-archive:${OCI_ARCHIVE}" | sha256sum | awk '{print $1}')"
prepared_digest="sha256:${prepared_digest}"
inspect_error="$(mktemp)"
trap 'rm -f "${inspect_error}"' EXIT

if remote_digest="$("${SKOPEO[@]}" inspect --format '{{.Digest}}' "docker://${IMAGE}" 2>"${inspect_error}")"; then
  if [ "${remote_digest}" != "${prepared_digest}" ]; then
    echo "Error: ${IMAGE} already points to ${remote_digest}, not ${prepared_digest}" >&2
    exit 1
  fi
  echo "* Reusing matching image ${IMAGE}@${remote_digest}"
elif grep -Eqi 'manifest unknown|name unknown|not found' "${inspect_error}"; then
  echo "* Publishing new image ${IMAGE}"
  "${SKOPEO[@]}" copy --all "oci-archive:${OCI_ARCHIVE}" "docker://${IMAGE}"
else
  echo "Error: could not determine whether ${IMAGE} exists" >&2
  cat "${inspect_error}" >&2
  exit 1
fi

published_digest="$("${SKOPEO[@]}" inspect --format '{{.Digest}}' "docker://${IMAGE}")"
if [ "${published_digest}" != "${prepared_digest}" ]; then
  echo "Error: published digest ${published_digest} does not match ${prepared_digest}" >&2
  exit 1
fi
printf '%s\n' "${published_digest}"
