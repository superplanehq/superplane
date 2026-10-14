#!/usr/bin/env bash

set -euo pipefail

if [ "$#" -ne 2 ]; then
  echo "Usage: release/fleet-manager/publish-chart.sh <version> <chart-archive>" >&2
  exit 1
fi

VERSION="$1"
CHART_ARCHIVE="$2"
CHART_REPOSITORY="${FLEET_MANAGER_CHART_REPO:-oci://ghcr.io/superplanehq/fleet-manager-chart}"
CHART_NAMESPACE="${CHART_REPOSITORY%/fleet-manager-chart}"
CHART_VERSION="${VERSION#v}"

node "$(dirname "$0")/../component-release.js" validate-version "${VERSION}" >/dev/null
if [ ! -f "${CHART_ARCHIVE}" ]; then
  echo "Error: chart archive does not exist: ${CHART_ARCHIVE}" >&2
  exit 1
fi

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
  if ! cmp -s "${CHART_ARCHIVE}" "${pull_dir}/fleet-manager-chart-${CHART_VERSION}.tgz"; then
    echo "Error: ${CHART_REPOSITORY}:${CHART_VERSION} already contains different content" >&2
    exit 1
  fi
  echo "Error: chart version ${CHART_VERSION} already exists; chart versions are immutable" >&2
  exit 1
fi
if ! grep -Eqi 'not found|manifest unknown|does not exist' "${inspect_error}"; then
  echo "Error: could not determine whether chart version ${CHART_VERSION} exists" >&2
  cat "${inspect_error}" >&2
  exit 1
fi

push_output="$(helm push "${CHART_ARCHIVE}" "${CHART_NAMESPACE}")"
digest="$(printf '%s\n' "${push_output}" | awk '$1 == "Digest:" {print $2}')"
if [[ ! "${digest}" =~ ^sha256:[0-9a-f]{64}$ ]]; then
  echo "Error: Helm did not return a valid chart digest" >&2
  exit 1
fi

rm -rf "${pull_dir}"
mkdir -p "${pull_dir}"
helm pull "${CHART_REPOSITORY}" --version "${CHART_VERSION}" --destination "${pull_dir}"
if ! cmp -s "${CHART_ARCHIVE}" "${pull_dir}/fleet-manager-chart-${CHART_VERSION}.tgz"; then
  echo "Error: published chart content does not match the prepared archive" >&2
  exit 1
fi

chart_reference="${CHART_REPOSITORY#oci://}"
registry="${chart_reference%%/*}"
repository="${chart_reference#*/}"
if [ "${registry}" != "ghcr.io" ]; then
  echo "Error: public chart verification only supports ghcr.io" >&2
  exit 1
fi
anonymous_token="$(
  curl --fail --silent --show-error \
    "https://${registry}/token?scope=repository:${repository}:pull" | \
    node -e '
      const fs = require("node:fs");
      const token = JSON.parse(fs.readFileSync(0, "utf8")).token;
      if (!token) process.exit(1);
      process.stdout.write(token);
    '
)"
public_digest="$(
  curl --fail --silent --show-error --head \
    --header "Authorization: Bearer ${anonymous_token}" \
    --header 'Accept: application/vnd.oci.image.manifest.v1+json' \
    "https://${registry}/v2/${repository}/manifests/${CHART_VERSION}" | \
    tr -d '\r' | sed -n 's/^[Dd]ocker-[Cc]ontent-[Dd]igest: //p'
)"
if [ "${public_digest}" != "${digest}" ]; then
  echo "Error: public chart digest does not match the published digest" >&2
  exit 1
fi

if [ -n "${FLEET_MANAGER_CHART_DIGEST_OUTPUT:-}" ]; then
  printf '%s\n' "${digest}" >"${FLEET_MANAGER_CHART_DIGEST_OUTPUT}"
fi
printf '%s\n' "${digest}"
