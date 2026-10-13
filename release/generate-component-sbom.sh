#!/usr/bin/env bash

set -euo pipefail

if [ "$#" -ne 3 ]; then
  echo "Usage: release/generate-component-sbom.sh <component> <input-directory> <output-file>" >&2
  exit 1
fi

COMPONENT="$1"
INPUT_DIRECTORY="$(cd "$2" && pwd)"
OUTPUT_FILE="$3"
OUTPUT_DIRECTORY="$(cd "$(dirname "${OUTPUT_FILE}")" && pwd)"
OUTPUT_NAME="$(basename "${OUTPUT_FILE}")"
SYFT_IMAGE="anchore/syft:v1.36.0@sha256:6733fa6ba7fb102d5b8eecae0e9ee7ee7091e613b8ce8d1fc9e6641335ab3962"

case "${COMPONENT}" in
runner | fleet-manager) ;;
*)
  echo "Error: component must be runner or fleet-manager" >&2
  exit 1
  ;;
esac

docker run --rm \
  --user "$(id -u):$(id -g)" \
  --env HOME=/tmp \
  --env SYFT_CHECK_FOR_APP_UPDATE=false \
  --volume "${INPUT_DIRECTORY}:/scan:ro" \
  --volume "${OUTPUT_DIRECTORY}:/output" \
  "${SYFT_IMAGE}" \
  scan dir:/scan \
  --source-name "${COMPONENT}" \
  --output "spdx-json=/output/${OUTPUT_NAME}"
