#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

if [ "${1-}" = "" ]; then
  echo "Usage: release/superplane-helm-chart/build.sh <version>"
  echo ""
  echo "Example:"
  echo "  release/superplane-helm-chart/build.sh v1.2.3"
  exit 1
fi

VERSION="$1"

# Remove 'v' prefix if present for chart version
CHART_VERSION="${VERSION#v}"

echo "* Packaging Helm chart for version ${VERSION}"

CHART_DIR="release/superplane-helm-chart/helm"
CHART_NAME="superplane-chart"
REGISTRY="oci://ghcr.io/superplanehq"

# Update Helm dependencies
echo "* Updating Helm dependencies"
helm dependency update "${CHART_DIR}"

# Package the chart with the specified version
helm package "${CHART_DIR}" --version "${CHART_VERSION}" --app-version "${CHART_VERSION}"

# Get the generated chart filename (Helm uses the chart name from Chart.yaml)
CHART_FILE="${CHART_NAME}-${CHART_VERSION}.tgz"

if [ ! -f "${CHART_FILE}" ]; then
  echo "Error: Chart file ${CHART_FILE} was not created"
  exit 1
fi

echo "* Chart packaged: ${CHART_FILE}"

echo "* Pushing chart to ${REGISTRY}"
helm push "${CHART_FILE}" "${REGISTRY}"

echo "* Cleaning up local chart file"
rm -f "${CHART_FILE}"

echo ""
echo "Done."
echo "Chart pushed to: ${REGISTRY}/${CHART_NAME}:${CHART_VERSION}"

