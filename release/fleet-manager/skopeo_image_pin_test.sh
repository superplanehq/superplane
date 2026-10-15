#!/usr/bin/env bash

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
PIPELINE="${REPO_ROOT}/.semaphore/publish-fleet-manager-release.yml"
PIN_PATTERN='^quay\.io/skopeo/stable:[A-Za-z0-9._]+-immutable@sha256:[0-9a-f]{64}$'

read_pin() {
  local file="$1"
  awk '
    $0 ~ /^[[:space:]]*#/ { next }
    $1 == "-" && $2 == "name:" && $3 == "SKOPEO_CONTAINER_IMAGE" { found = 1; next }
    found && $1 == "value:" {
      value = $2
      gsub(/^["'\'']|["'\'']$/, "", value)
      print value
      exit
    }
  ' "${file}"
}

check_pin_file() {
  local file="$1"
  local pin
  pin="$(read_pin "${file}")"
  if [[ ! "${pin}" =~ ${PIN_PATTERN} ]]; then
    echo "SKOPEO_CONTAINER_IMAGE must be quay.io/skopeo/stable:<tag>-immutable@sha256:<64 hex>, got: ${pin:-<missing>}" >&2
    return 1
  fi
}

check_pin_file "${PIPELINE}"

TEST_DIRECTORY="$(mktemp -d)"
trap 'rm -rf "${TEST_DIRECTORY}"' EXIT
OLD_PIN="quay.io/skopeo/stable:v1.22.3@sha256:22ea3c3baf639147b07fb9f2ea28a6308edf7909e919d91a5d296af0121d7ed9"
cat >"${TEST_DIRECTORY}/pipeline.yml" <<EOF
global_job_config:
  env_vars:
    - name: SKOPEO_CONTAINER_IMAGE
      value: ${OLD_PIN}
EOF

if check_pin_file "${TEST_DIRECTORY}/pipeline.yml" >/dev/null 2>&1; then
  echo "Expected the old mutable tag pin to fail" >&2
  exit 1
fi

echo "Fleet Manager skopeo image pin tests passed."
