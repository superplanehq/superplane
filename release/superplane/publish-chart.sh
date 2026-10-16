#!/usr/bin/env bash

set -euo pipefail

if [ "$#" -ne 2 ]; then
  echo "Usage: release/superplane/publish-chart.sh <version> <chart-archive>" >&2
  exit 1
fi

VERSION="$1"
CHART_ARCHIVE="$2"
CHART_REPOSITORY="${SUPERPLANE_CHART_REPO:-oci://ghcr.io/superplanehq/superplane-chart}"
CHART_NAMESPACE="${CHART_REPOSITORY%/superplane-chart}"
CHART_VERSION="${VERSION#v}"

public_digest() {
  local chart_reference registry repository token
  chart_reference="${CHART_REPOSITORY#oci://}"
  registry="${chart_reference%%/*}"
  repository="${chart_reference#*/}"
  if [ "${registry}" != "ghcr.io" ]; then
    echo "Error: public chart verification only supports ghcr.io" >&2
    return 1
  fi
  token="$(
    curl --fail --silent --show-error \
      "https://${registry}/token?scope=repository:${repository}:pull" | \
      node -e 'const value = JSON.parse(require("node:fs").readFileSync(0, "utf8")).token; if (!value) process.exit(1); process.stdout.write(value)'
  )"
  curl --fail --silent --show-error --head \
    --header "Authorization: Bearer ${token}" \
    --header 'Accept: application/vnd.oci.image.manifest.v1+json' \
    "https://${registry}/v2/${repository}/manifests/${CHART_VERSION}" | \
    tr -d '\r' | sed -n 's/^[Dd]ocker-[Cc]ontent-[Dd]igest: //p'
}

node "$(dirname "$0")/../component-release.js" validate-version "${VERSION}" >/dev/null
test -f "${CHART_ARCHIVE}"
actual_version="$(helm show chart "${CHART_ARCHIVE}" | awk '$1 == "version:" {print $2}')"
actual_app_version="$(helm show chart "${CHART_ARCHIVE}" | awk '$1 == "appVersion:" {print $2}')"
if [ "${actual_version}" != "${CHART_VERSION}" ] || [ "${actual_app_version}" != "${VERSION}" ]; then
  echo "Error: chart metadata does not match ${VERSION}" >&2
  exit 1
fi

pull_dir="$(mktemp -d)"
inspect_error="$(mktemp)"
trap 'rm -rf "${pull_dir}"; rm -f "${inspect_error}"' EXIT
if helm pull "${CHART_REPOSITORY}" --version "${CHART_VERSION}" --destination "${pull_dir}" 2>"${inspect_error}"; then
  if ! cmp -s "${CHART_ARCHIVE}" "${pull_dir}/superplane-chart-${CHART_VERSION}.tgz"; then
    echo "Error: chart version ${CHART_VERSION} already contains different content" >&2
    exit 1
  fi
  digest="$(public_digest)"
  if [[ ! "${digest}" =~ ^sha256:[0-9a-f]{64}$ ]]; then
    echo "Error: registry did not return a valid chart digest" >&2
    exit 1
  fi
  echo "* Reusing matching chart version ${CHART_VERSION}" >&2
  printf '%s\n' "${digest}"
  exit 0
fi
if ! grep -Eqi 'not found|manifest unknown|does not exist' "${inspect_error}"; then
  echo "Error: could not determine whether chart version ${CHART_VERSION} exists" >&2
  cat "${inspect_error}" >&2
  exit 1
fi

push_output="$(helm push "${CHART_ARCHIVE}" "${CHART_NAMESPACE}" 2>&1)"
digest="$(printf '%s\n' "${push_output}" | awk '$1 == "Digest:" {print $2}')"
if [[ ! "${digest}" =~ ^sha256:[0-9a-f]{64}$ ]]; then
  echo "Error: Helm did not return a valid chart digest" >&2
  exit 1
fi

rm -rf "${pull_dir}"
mkdir -p "${pull_dir}"
helm pull "${CHART_REPOSITORY}" --version "${CHART_VERSION}" --destination "${pull_dir}"
cmp -s "${CHART_ARCHIVE}" "${pull_dir}/superplane-chart-${CHART_VERSION}.tgz"
public_chart_digest="$(public_digest)"
if [ "${public_chart_digest}" != "${digest}" ]; then
  echo "Error: public chart digest does not match the published digest" >&2
  exit 1
fi
printf '%s\n' "${digest}"
