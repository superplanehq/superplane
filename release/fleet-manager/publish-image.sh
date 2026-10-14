#!/usr/bin/env bash

set -euo pipefail

if [ "$#" -ne 2 ]; then
  echo "Usage: release/fleet-manager/publish-image.sh <version> <oci-archive>" >&2
  exit 1
fi

VERSION="$1"
OCI_ARCHIVE="$2"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
IMAGE_REPOSITORY="${FLEET_MANAGER_IMAGE_REPO:-ghcr.io/superplanehq/fleet-manager}"
IMAGE="${IMAGE_REPOSITORY}:${VERSION}"
SKOPEO=(bash "${SCRIPT_DIR}/run-skopeo.sh")

node "${SCRIPT_DIR}/../component-release.js" validate-version "${VERSION}" >/dev/null
if [ ! -f "${OCI_ARCHIVE}" ]; then
  echo "Error: OCI archive does not exist: ${OCI_ARCHIVE}" >&2
  exit 1
fi
if [ -z "${SKOPEO_CONTAINER_IMAGE:-}" ] && ! command -v skopeo >/dev/null 2>&1; then
  echo "Error: skopeo is required to publish the prepared OCI archive" >&2
  exit 1
fi

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
elif grep -Eqi 'manifest unknown' "${inspect_error}"; then
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
if [ -n "${FLEET_MANAGER_DIGEST_OUTPUT:-}" ]; then
  printf '%s\n' "${published_digest}" >"${FLEET_MANAGER_DIGEST_OUTPUT}"
fi
printf '%s\n' "${published_digest}"
