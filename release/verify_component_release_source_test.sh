#!/usr/bin/env bash

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERIFY_SCRIPT="${REPO_ROOT}/release/verify-component-release-source.sh"
SOURCE_SHA="aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

export PATH="${TMP_DIR}:${PATH}"
export SOURCE_SHA
export GIT_CALLS="${TMP_DIR}/git-calls"

cat >"${TMP_DIR}/git" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

if [ "${1:-}" = "-C" ]; then
  shift 2
fi

printf '%s\n' "$*" >>"${GIT_CALLS}"
case "${1:-}" in
  rev-parse)
    printf '%s\n' "${SOURCE_SHA}"
    ;;
  status|fetch)
    ;;
  merge-base)
    ;;
  ls-remote)
    exit 2
    ;;
  *)
    echo "unexpected git command: $*" >&2
    exit 1
    ;;
esac
EOF
chmod +x "${TMP_DIR}/git"

cat >"${TMP_DIR}/curl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

output=""
url=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --output)
      output="$2"
      shift 2
      ;;
    --header|--write-out)
      shift 2
      ;;
    --fail|--silent|--show-error)
      shift
      ;;
    *)
      url="$1"
      shift
      ;;
  esac
done

case "${url}" in
  */commits/*/status)
    printf '%s\n' '{"statuses":[{"context":"ci/semaphoreci/push: CI","state":"success"}]}'
    ;;
  */releases/tags/*)
    printf '%s\n' '{}' >"${output}"
    printf '404'
    ;;
  *)
    echo "unexpected curl URL: ${url}" >&2
    exit 1
    ;;
esac
EOF
chmod +x "${TMP_DIR}/curl"

verify_source() {
  local component="$1"
  local branch="$2"

  SEMAPHORE_GIT_SHA="${SOURCE_SHA}" \
    SEMAPHORE_GIT_BRANCH="${branch}" \
    SEMAPHORE_PIPELINE_PROMOTION=true \
    SEMAPHORE_WORKFLOW_ID=workflow-1 \
    SEMAPHORE_PIPELINE_ID=pipeline-1 \
    bash "${VERIFY_SCRIPT}" "${component}" v1.0.0-rc1
}

: >"${GIT_CALLS}"
verify_source superplane release/v1.0.x >"${TMP_DIR}/release-branch-stdout"
grep -Fqx 'fetch --quiet --tags origin release/v1.0.x' "${GIT_CALLS}"
grep -Fqx \
  "merge-base --is-ancestor ${SOURCE_SHA} origin/release/v1.0.x" \
  "${GIT_CALLS}"

: >"${GIT_CALLS}"
verify_source runner main >"${TMP_DIR}/main-stdout"
grep -Fqx 'fetch --quiet --tags origin main' "${GIT_CALLS}"
grep -Fqx \
  "merge-base --is-ancestor ${SOURCE_SHA} origin/main" \
  "${GIT_CALLS}"

if verify_source runner release/v1.0.x >"${TMP_DIR}/runner-stdout" 2>"${TMP_DIR}/runner-stderr"; then
  echo "Runner release unexpectedly accepted a release branch" >&2
  exit 1
fi
grep -q 'must originate from main' "${TMP_DIR}/runner-stderr"

if verify_source superplane release/v1.x >"${TMP_DIR}/invalid-stdout" 2>"${TMP_DIR}/invalid-stderr"; then
  echo "SuperPlane release unexpectedly accepted an invalid release branch" >&2
  exit 1
fi
grep -q 'must originate from main or release/vMAJOR.MINOR.x' "${TMP_DIR}/invalid-stderr"

echo "Component release source tests passed."
