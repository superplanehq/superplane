#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

if [ "$#" -ne 2 ]; then
  echo "Usage: release/superplane-helm-chart/build.sh <version> <output-directory>"
  echo ""
  echo "Examples:"
  echo "  release/superplane-helm-chart/build.sh v1.2.3 build/superplane/v1.2.3"
  exit 1
fi

VERSION="$1"
OUTPUT_DIR="$2"

# Chart version drops the leading v. appVersion stays the image tag.
CHART_VERSION="${VERSION#v}"
APP_VERSION="${VERSION}"

echo "* Packaging Helm chart for version ${VERSION}"

CHART_DIR="release/superplane-helm-chart/helm"
CHART_NAME="superplane-chart"
mkdir -p "${OUTPUT_DIR}"

# Update Helm dependencies
echo "* Updating Helm dependencies"
helm dependency update "${CHART_DIR}"

if [ "${COMPONENT_RELEASES_ALREADY_RESOLVED:-}" = "yes" ]; then
  echo "* Using component release defaults resolved for this release"
else
  echo "* Resolving component release defaults"
  node release/superplane-helm-chart/update-component-release-defaults.js
fi

# Package the chart with the specified version
helm package "${CHART_DIR}" \
  --version "${CHART_VERSION}" \
  --app-version "${APP_VERSION}" \
  --destination "${OUTPUT_DIR}" >/dev/null

# Get the generated chart filename (Helm uses the chart name from Chart.yaml)
CHART_FILE="${CHART_NAME}-${CHART_VERSION}.tgz"
RELEASE_FILE="superplane-${CHART_VERSION}.tgz"

if [ ! -f "${OUTPUT_DIR}/${CHART_FILE}" ]; then
  echo "Error: Chart file ${CHART_FILE} was not created"
  exit 1
fi

test "$(helm show chart "${OUTPUT_DIR}/${CHART_FILE}" | awk '$1 == "version:" {print $2}')" = "${CHART_VERSION}"
test "$(helm show chart "${OUTPUT_DIR}/${CHART_FILE}" | awk '$1 == "appVersion:" {print $2}')" = "${APP_VERSION}"
mv "${OUTPUT_DIR}/${CHART_FILE}" "${OUTPUT_DIR}/${RELEASE_FILE}"

echo "* Chart packaged: ${OUTPUT_DIR}/${RELEASE_FILE}"

echo ""
echo "Done."
echo "Chart: ${OUTPUT_DIR}/${RELEASE_FILE}"
