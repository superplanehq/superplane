#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

cat >"${TMP_DIR}/curl" <<'SHIM'
#!/usr/bin/env sh
set -eu

method=GET
url=""
payload=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --request)
      method="$2"
      shift 2
      ;;
    --data-binary)
      payload="$2"
      shift 2
      ;;
    http*)
      url="$1"
      shift
      ;;
    *)
      shift
      ;;
  esac
done

printf '%s %s %s\n' "$method" "$url" "$payload" >>"${CURL_LOG}"
if [ "${CURL_FAIL_ONCE:-}" = yes ] && [ ! -e "${CURL_FAILURE_MARKER}" ]; then
  touch "${CURL_FAILURE_MARKER}"
  exit 7
fi

case "${method}" in
  GET)
    printf '%s' "${GET_STATUS}"
    ;;
  POST)
    printf '%s' "${POST_STATUS:-201}"
    ;;
  PATCH)
    printf '%s' "${PATCH_STATUS:-200}"
    ;;
esac
SHIM
chmod +x "${TMP_DIR}/curl"

cat >"${TMP_DIR}/sleep" <<'SHIM'
#!/usr/bin/env sh
exit 0
SHIM
chmod +x "${TMP_DIR}/sleep"

export PATH="${TMP_DIR}:${PATH}"
export CURL_LOG="${TMP_DIR}/curl.log"
export CURL_FAILURE_MARKER="${TMP_DIR}/curl-failed"
export SUPERPLANE_URL="https://superplane.example.com"
export INSTALLATION_ADMIN_TOKEN="must-not-appear-in-output"

create_payload='{"fleetId":"e1-large-amd64","runnerVersion":"v0.1.0","spec":{"operatingSystem":"linux"}}'
update_payload='{"runnerVersion":"v0.1.0","spec":{"operatingSystem":"linux"}}'
create_b64="$(printf '%s' "${create_payload}" | base64 | tr -d '\n')"
update_b64="$(printf '%s' "${update_payload}" | base64 | tr -d '\n')"

GET_STATUS=404 POST_STATUS=201 \
  sh "${SCRIPT_DIR}/helm/files/sync-fleets.sh" \
  e1-large-amd64 "${create_b64}" "${update_b64}"
grep -q '^GET .*e1-large-amd64 ' "${CURL_LOG}"
grep -q '^POST .*installation/fleets ' "${CURL_LOG}"

: >"${CURL_LOG}"
GET_STATUS=200 PATCH_STATUS=200 \
  sh "${SCRIPT_DIR}/helm/files/sync-fleets.sh" \
  e1-large-amd64 "${create_b64}" "${update_b64}"
grep -q '^PATCH .*e1-large-amd64 ' "${CURL_LOG}"

: >"${CURL_LOG}"
rm -f "${CURL_FAILURE_MARKER}"
CURL_FAIL_ONCE=yes GET_STATUS=200 PATCH_STATUS=200 \
  sh "${SCRIPT_DIR}/helm/files/sync-fleets.sh" \
  e1-large-amd64 "${create_b64}" "${update_b64}"
if [ "$(grep -c '^GET ' "${CURL_LOG}")" -ne 2 ]; then
  echo "A temporary connection failure must retry the request" >&2
  exit 1
fi
if grep -q '^POST ' "${CURL_LOG}"; then
  echo "An existing fleet must not be created again" >&2
  exit 1
fi

: >"${CURL_LOG}"
GET_STATUS=404 POST_STATUS=409 PATCH_STATUS=200 \
  sh "${SCRIPT_DIR}/helm/files/sync-fleets.sh" \
  e1-large-amd64 "${create_b64}" "${update_b64}"
grep -q '^POST .*installation/fleets ' "${CURL_LOG}"
grep -q '^PATCH .*e1-large-amd64 ' "${CURL_LOG}"

if GET_STATUS=401 sh "${SCRIPT_DIR}/helm/files/sync-fleets.sh" \
  e1-large-amd64 "${create_b64}" "${update_b64}" \
  >"${TMP_DIR}/stdout" 2>"${TMP_DIR}/stderr"; then
  echo "Authentication failures must fail the sync" >&2
  exit 1
fi
if grep -R -q "${INSTALLATION_ADMIN_TOKEN}" "${TMP_DIR}/stdout" "${TMP_DIR}/stderr"; then
  echo "The sync script wrote the bearer token to output" >&2
  exit 1
fi

echo "Fleet sync tests passed."
