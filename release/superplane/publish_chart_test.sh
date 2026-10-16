#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

printf 'prepared chart' >"${TMP_DIR}/superplane-1.2.3-rc1.tgz"
cat >"${TMP_DIR}/helm" <<'SHIM'
#!/usr/bin/env bash
set -euo pipefail
case "$1 $2" in
  "show chart") printf 'version: 1.2.3-rc1\nappVersion: v1.2.3-rc1\n' ;;
  "pull oci://ghcr.io/superplanehq/superplane-chart")
    destination=""
    while [ "$#" -gt 0 ]; do
      if [ "$1" = --destination ]; then destination="$2"; break; fi
      shift
    done
    count="$(cat "${PULL_COUNT}" 2>/dev/null || printf 0)"
    count="$((count + 1))"
    printf '%s' "${count}" >"${PULL_COUNT}"
    if [ "${PULL_MODE}" = missing-once ] && [ "${count}" -eq 1 ]; then
      echo 'manifest unknown' >&2
      exit 1
    fi
    if [ "${PULL_MODE}" = content-mismatch ] && [ "${count}" -eq 1 ]; then
      echo 'manifest unknown' >&2
      exit 1
    fi
    if [ "${PULL_MODE}" = content-mismatch ]; then
      printf 'different' >"${destination}/superplane-chart-1.2.3-rc1.tgz"
      exit 0
    fi
    cp "${REMOTE_CHART}" "${destination}/superplane-chart-1.2.3-rc1.tgz"
    ;;
  "push "*) printf 'push\n' >>"${PUSH_LOG}"; printf 'Digest: sha256:%064d\n' 1 ;;
  *) echo "unexpected helm invocation: $*" >&2; exit 1 ;;
esac
SHIM
cat >"${TMP_DIR}/curl" <<'SHIM'
#!/usr/bin/env bash
set -euo pipefail
if [[ "$*" == *'/token?scope=repository:superplanehq/superplane-chart:pull'* ]]; then
  printf '{"token":"token"}'
elif [[ "$*" == *'/v2/superplanehq/superplane-chart/manifests/1.2.3-rc1'* ]]; then
  printf 'Docker-Content-Digest: sha256:%064d\r\n' 1
else
  exit 1
fi
SHIM
chmod +x "${TMP_DIR}/helm" "${TMP_DIR}/curl"

export PATH="${TMP_DIR}:${PATH}"
export PULL_COUNT="${TMP_DIR}/pull-count"
export PUSH_LOG="${TMP_DIR}/push.log"
export REMOTE_CHART="${TMP_DIR}/remote.tgz"
cp "${TMP_DIR}/superplane-1.2.3-rc1.tgz" "${REMOTE_CHART}"

PULL_MODE=existing bash "${SCRIPT_DIR}/publish-chart.sh" \
  v1.2.3-rc1 "${TMP_DIR}/superplane-1.2.3-rc1.tgz" >"${TMP_DIR}/digest"
grep -Eq '^sha256:[0-9a-f]{64}$' "${TMP_DIR}/digest"

printf 'different' >"${REMOTE_CHART}"
: >"${PULL_COUNT}"
: >"${PUSH_LOG}"
PULL_MODE=existing bash "${SCRIPT_DIR}/publish-chart.sh" \
  v1.2.3-rc1 "${TMP_DIR}/superplane-1.2.3-rc1.tgz" >"${TMP_DIR}/digest"
grep -Eq '^sha256:[0-9a-f]{64}$' "${TMP_DIR}/digest"
test ! -s "${PUSH_LOG}"

cp "${TMP_DIR}/superplane-1.2.3-rc1.tgz" "${REMOTE_CHART}"
: >"${PULL_COUNT}"
PULL_MODE=missing-once bash "${SCRIPT_DIR}/publish-chart.sh" \
  v1.2.3-rc1 "${TMP_DIR}/superplane-1.2.3-rc1.tgz" >"${TMP_DIR}/digest"
grep -Eq '^sha256:[0-9a-f]{64}$' "${TMP_DIR}/digest"

: >"${PULL_COUNT}"
if PULL_MODE=content-mismatch bash "${SCRIPT_DIR}/publish-chart.sh" \
  v1.2.3-rc1 "${TMP_DIR}/superplane-1.2.3-rc1.tgz" \
  >"${TMP_DIR}/digest" 2>"${TMP_DIR}/stderr"; then
  echo "A published chart with different content must fail" >&2
  exit 1
fi
grep -q 'published chart content does not match' "${TMP_DIR}/stderr"

echo "SuperPlane chart publication tests passed."
