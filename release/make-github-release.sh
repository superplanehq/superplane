#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<EOF
Usage: release/make-github-release.sh <version>

Environment variables:
  GITHUB_TOKEN       GitHub token with repo permissions (required)

Example:
  GITHUB_TOKEN=... release/make-github-release.sh 1.0.0
EOF
}

if [[ "${1-}" == "" ]]; then
  usage
  exit 1
fi

VERSION="$1"

if [[ "${VERSION}" == "" || "${#VERSION}" -lt 2 ]]; then
  echo "Error: Version is required and must be at least 2 characters." >&2
  exit 1
fi

if [[ -z "${GITHUB_TOKEN-}" ]]; then
  echo "Error: GITHUB_TOKEN is required." >&2
  exit 1
fi

GITHUB_REPOSITORY="superplanehq/superplane"

BUILD_ROOT="build/single-host-${VERSION}"
ARTIFACT_PATH="${BUILD_ROOT}/superplane-single-host.tar.gz"
ASSET_NAME="superplane-single-host.tar.gz"

if [[ ! -f "${ARTIFACT_PATH}" ]]; then
  echo "Error: ${ARTIFACT_PATH} does not exist. Run release/single-host/build.sh ${VERSION} first." >&2
  exit 1
fi

API_BASE="https://api.github.com"
UPLOAD_BASE="https://uploads.github.com"
USER_AGENT="superplane-release-script"
API_VERSION="2022-11-28"

echo "* Creating GitHub release for version ${VERSION}"

create_release_payload() {
  cat <<EOF
{"tag_name":"${VERSION}","target_commitish":"main","name":"${VERSION}","body":"Release ${VERSION}","draft":false,"prerelease":false,"generate_release_notes":false}
EOF
}

payload="$(create_release_payload)"

response=$(curl -sS -w "\n%{http_code}" \
  -X POST \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer ${GITHUB_TOKEN}" \
  -H "User-Agent: ${USER_AGENT}" \
  -H "X-GitHub-Api-Version: ${API_VERSION}" \
  "${API_BASE}/repos/${GITHUB_REPOSITORY}/releases" \
  -d "${payload}")

http_body=$(printf '%s\n' "${response}" | sed '$d')
http_code=$(printf '%s\n' "${response}" | tail -n1)
if [[ "${http_code}" != "201" ]]; then
  echo "Error: Failed to create release (HTTP ${http_code}). Response:" >&2
  echo "${http_body}" >&2
  exit 1
fi

release_id=$(printf '%s\n' "${http_body}" | python3 - <<'EOF' || true
import sys, json
try:
    data = json.load(sys.stdin)
    rid = data.get("id")
    print(rid if rid is not None else "")
except Exception:
    print("")
EOF
)

if [[ -z "${release_id}" ]]; then
  echo "Error: Could not parse release ID from response." >&2
  echo "${http_body}" >&2
  exit 1
fi

echo "* Release created successfully (id: ${release_id})"
echo "* Uploading asset ${ASSET_NAME} from ${ARTIFACT_PATH}"

upload_response=$(curl -sS -w "\n%{http_code}" \
  -X POST \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer ${GITHUB_TOKEN}" \
  -H "User-Agent: ${USER_AGENT}" \
  -H "X-GitHub-Api-Version: ${API_VERSION}" \
  -H "Content-Type: application/octet-stream" \
  --data-binary @"${ARTIFACT_PATH}" \
  "${UPLOAD_BASE}/repos/${GITHUB_REPOSITORY}/releases/${release_id}/assets?name=${ASSET_NAME}")

upload_body=$(printf '%s\n' "${upload_response}" | sed '$d')
upload_code=$(printf '%s\n' "${upload_response}" | tail -n1)

if [[ "${upload_code}" != "201" ]]; then
  echo "Error: Failed to upload asset (HTTP ${upload_code}). Response:" >&2
  echo "${upload_body}" >&2
  exit 1
fi

echo "* ${ASSET_NAME} uploaded successfully"
echo "Done."
