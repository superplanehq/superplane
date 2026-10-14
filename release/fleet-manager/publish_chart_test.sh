#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

printf 'prepared chart' >"${TMP_DIR}/fleet-manager-chart-1.2.3.tgz"

cat >"${TMP_DIR}/helm" <<'SHIM'
#!/usr/bin/env bash
set -euo pipefail

case "$1 $2" in
  "show chart")
    printf 'version: 1.2.3\nappVersion: v1.2.3\n'
    ;;
  "pull oci://ghcr.io/superplanehq/fleet-manager-chart")
    destination=""
    while [ "$#" -gt 0 ]; do
      if [ "$1" = "--destination" ]; then
        destination="$2"
        break
      fi
      shift
    done
    count="$(cat "${PULL_COUNT}" 2>/dev/null || printf 0)"
    count="$((count + 1))"
    printf '%s' "${count}" >"${PULL_COUNT}"
    if [ "${PULL_MODE}" = "missing-once" ] && [ "${count}" -eq 1 ]; then
      echo 'manifest unknown' >&2
      exit 1
    fi
    cp "${REMOTE_CHART}" "${destination}/fleet-manager-chart-1.2.3.tgz"
    ;;
  "push "*)
    printf 'Pushed: test\nDigest: sha256:%064d\n' 1 >&2
    ;;
  *)
    echo "unexpected helm invocation: $*" >&2
    exit 1
    ;;
esac
SHIM
chmod +x "${TMP_DIR}/helm"

cat >"${TMP_DIR}/curl" <<'SHIM'
#!/usr/bin/env bash
set -euo pipefail

if [[ "$*" == *"/token?scope=repository:superplanehq/fleet-manager-chart:pull"* ]]; then
  printf '{"token":"anonymous-token"}'
  exit 0
fi
if [[ "$*" == *"/v2/superplanehq/fleet-manager-chart/manifests/1.2.3"* ]]; then
  printf 'HTTP/2 200\r\nDocker-Content-Digest: sha256:%064d\r\n\r\n' 1
  exit 0
fi
echo "unexpected curl invocation: $*" >&2
exit 1
SHIM
chmod +x "${TMP_DIR}/curl"

export PATH="${TMP_DIR}:${PATH}"
export PULL_COUNT="${TMP_DIR}/pull-count"
export REMOTE_CHART="${TMP_DIR}/remote.tgz"

cp "${TMP_DIR}/fleet-manager-chart-1.2.3.tgz" "${REMOTE_CHART}"
export PULL_MODE=existing
if bash "${SCRIPT_DIR}/publish-chart.sh" v1.2.3 "${TMP_DIR}/fleet-manager-chart-1.2.3.tgz" \
  >"${TMP_DIR}/stdout" 2>"${TMP_DIR}/stderr"; then
  echo "An existing chart version must be refused" >&2
  exit 1
fi
grep -q 'chart version 1.2.3 already exists' "${TMP_DIR}/stderr"

printf 'different chart' >"${REMOTE_CHART}"
: >"${PULL_COUNT}"
if bash "${SCRIPT_DIR}/publish-chart.sh" v1.2.3 "${TMP_DIR}/fleet-manager-chart-1.2.3.tgz" \
  >"${TMP_DIR}/stdout" 2>"${TMP_DIR}/stderr"; then
  echo "A chart collision must fail" >&2
  exit 1
fi
grep -q 'already contains different content' "${TMP_DIR}/stderr"

cp "${TMP_DIR}/fleet-manager-chart-1.2.3.tgz" "${REMOTE_CHART}"
: >"${PULL_COUNT}"
export PULL_MODE=missing-once
export FLEET_MANAGER_CHART_DIGEST_OUTPUT="${TMP_DIR}/digest"
bash "${SCRIPT_DIR}/publish-chart.sh" v1.2.3 "${TMP_DIR}/fleet-manager-chart-1.2.3.tgz" >/dev/null
grep -Eq '^sha256:[0-9a-f]{64}$' "${TMP_DIR}/digest"

echo "Fleet Manager chart publication tests passed."
