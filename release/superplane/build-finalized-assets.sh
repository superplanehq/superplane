#!/usr/bin/env bash

set -euo pipefail

if [ "$#" -ne 4 ]; then
  echo "Usage: release/superplane/build-finalized-assets.sh <candidate-values> <version> <source-sha> <output-directory>" >&2
  exit 1
fi

CANDIDATE_VALUES="$1"
VERSION="$2"
SOURCE_SHA="$3"
OUTPUT_DIR="$4"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
TARGET_VALUES="${REPO_ROOT}/release/superplane-helm-chart/helm/values.yaml"

if [ ! -f "${CANDIDATE_VALUES}" ]; then
  echo "Error: candidate chart values not found: ${CANDIDATE_VALUES}" >&2
  exit 1
fi

node "${REPO_ROOT}/release/superplane-helm-chart/update-component-release-defaults.js" \
  reuse \
  "${CANDIDATE_VALUES}" \
  "${TARGET_VALUES}"

COMPONENT_RELEASES_ALREADY_RESOLVED=yes \
  bash "${SCRIPT_DIR}/build-assets.sh" \
  "${VERSION}" \
  "${SOURCE_SHA}" \
  "${OUTPUT_DIR}"
