#!/usr/bin/env bash

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
UPDATER="${REPO_ROOT}/release/superplane-helm-chart/update-component-release-defaults.js"
BUILD_SCRIPT="${REPO_ROOT}/release/superplane-helm-chart/build.sh"
VALUES_PATH="${REPO_ROOT}/release/superplane-helm-chart/helm/values.yaml"
API_ROOT="https://api.github.com/repos/superplanehq/superplane/releases?per_page=100"
PAGE_TWO="${API_ROOT}&page=2"
FALLBACK_COMMENT="# Saved versions are the fallback when release lookup fails."

TMP_DIR="$(mktemp -d)"
VALUES_BACKUP="${TMP_DIR}/values.yaml.bak"
cp "${VALUES_PATH}" "${VALUES_BACKUP}"
trap 'cp "${VALUES_BACKUP}" "${VALUES_PATH}"; rm -rf "${TMP_DIR}"' EXIT

unset GITHUB_TOKEN
export PATH="${TMP_DIR}:${PATH}"
export TMP_DIR
export API_ROOT
export PAGE_TWO
export CURL_SEQUENCE="${TMP_DIR}/sequence"
export CURL_AUTH_FLAG="${TMP_DIR}/authorization"

cat >"${TMP_DIR}/curl" <<'SHIM'
#!/usr/bin/env bash
set -euo pipefail

output=""
header_file=""
url=""
authorization="absent"
while [ "$#" -gt 0 ]; do
  case "$1" in
    --output)
      output="$2"
      shift 2
      ;;
    --dump-header)
      header_file="$2"
      shift 2
      ;;
    --header)
      header_value="$2"
      if [[ "${header_value}" == @* ]]; then
        if grep -q '^Authorization:' "${header_value#@}"; then
          authorization="present"
        fi
      elif [[ "${header_value}" == Authorization:* ]]; then
        authorization="present"
      fi
      shift 2
      ;;
    --write-out)
      shift 2
      ;;
    --max-time)
      printf '%s' "$2" >"${TMP_DIR}/max-time"
      shift 2
      ;;
    --silent|--show-error)
      shift
      ;;
    --)
      url="$2"
      shift 2
      ;;
    *)
      if [[ "$1" == http:* || "$1" == https:* ]]; then
        url="$1"
      fi
      shift
      ;;
  esac
done

if [ -z "${output}" ] || [ -z "${header_file}" ] || [ -z "${url}" ]; then
  echo "curl shim expected --output, --dump-header, and a URL" >&2
  exit 1
fi

printf '%s\n' "${authorization}" >"${CURL_AUTH_FLAG}"
printf 'curl %s\n' "${url}" >>"${CURL_SEQUENCE}"

if [[ "${url}" == *"/releases/latest"* ]]; then
  echo "latest-release endpoint must not be used" >&2
  exit 1
fi

write_response() {
  local status="$1"
  local body="$2"
  local link="${3:-}"
  if [ -n "${link}" ]; then
    printf 'HTTP/1.1 %s\r\nLink: %s\r\n\r\n' "${status}" "${link}" >"${header_file}"
  else
    printf 'HTTP/1.1 %s\r\n\r\n' "${status}" >"${header_file}"
  fi
  printf '%s' "${body}" >"${output}"
  printf '%s' "${status%% *}"
}

page_one="$(cat "${TMP_DIR}/page-one.json" 2>/dev/null || true)"
page_two="$(cat "${TMP_DIR}/page-two.json" 2>/dev/null || true)"
next_link="<${PAGE_TWO}>; rel=\"next\""
last_link="<${API_ROOT}&page=9>; rel=\"last\""

case "${CURL_MODE}" in
  success)
    case "${url}" in
      "${API_ROOT}")
        write_response "200 OK" "${page_one}" "${last_link}, ${next_link}"
        ;;
      "${PAGE_TWO}")
        write_response "200 OK" "${page_two}"
        ;;
      *)
        echo "unexpected releases URL: ${url}" >&2
        exit 1
        ;;
    esac
    ;;
  numeric|fleet-manager-missing|runner-missing)
    if [ "${url}" != "${API_ROOT}" ]; then
      echo "unexpected releases URL: ${url}" >&2
      exit 1
    fi
    write_response "200 OK" "${page_one}"
    ;;
  http-error)
    if [ "${url}" != "${API_ROOT}" ]; then
      echo "unexpected releases URL: ${url}" >&2
      exit 1
    fi
    write_response "500 Internal Server Error" '{"message":"unavailable"}'
    ;;
  stall)
    if [ "${url}" != "${API_ROOT}" ]; then
      echo "unexpected releases URL: ${url}" >&2
      exit 1
    fi
    sleep 11
    exit 28
    ;;
  invalid-json)
    if [ "${url}" != "${API_ROOT}" ]; then
      echo "unexpected releases URL: ${url}" >&2
      exit 1
    fi
    write_response "200 OK" 'not-json'
    ;;
  page-two-error)
    case "${url}" in
      "${API_ROOT}")
        write_response "200 OK" "${page_one}" "${last_link}, ${next_link}"
        ;;
      "${PAGE_TWO}")
        write_response "502 Bad Gateway" '{"message":"unavailable"}'
        ;;
      *)
        echo "unexpected releases URL: ${url}" >&2
        exit 1
        ;;
    esac
    ;;
  cycle)
    case "${url}" in
      "${API_ROOT}")
        write_response "200 OK" "${page_one}" "<${PAGE_TWO}>; rel=\"next\""
        ;;
      "${PAGE_TWO}")
        write_response "200 OK" "${page_two}" "<${API_ROOT}>; rel=\"next\""
        ;;
      *)
        echo "unexpected releases URL: ${url}" >&2
        exit 1
        ;;
    esac
    ;;
  *)
    echo "unexpected curl mode: ${CURL_MODE}" >&2
    exit 1
    ;;
esac
SHIM
chmod +x "${TMP_DIR}/curl"

cat >"${TMP_DIR}/helm" <<'SHIM'
#!/usr/bin/env bash
set -euo pipefail

printf 'helm %s\n' "$1" >>"${CURL_SEQUENCE}"
case "$1" in
  dependency)
    ;;
  package)
    version=""
    destination=""
    while [ "$#" -gt 0 ]; do
      case "$1" in
        --version) version="$2"; shift 2 ;;
        --destination) destination="$2"; shift 2 ;;
        *) shift ;;
      esac
    done
    if [ -z "${version}" ] || [ -z "${destination}" ]; then
      echo "helm package did not receive its version and destination" >&2
      exit 1
    fi
    printf 'chart' >"${destination}/superplane-chart-${version}.tgz"
    ;;
  show)
    printf 'version: 1.2.3\nappVersion: v1.2.3\n'
    ;;
  *)
    echo "unexpected helm invocation: $*" >&2
    exit 1
    ;;
esac
SHIM
chmod +x "${TMP_DIR}/helm"

write_saved_values() {
  cat >"${TMP_DIR}/values.yaml" <<'EOF'
other: keep

componentReleases:
  runnerVersion: v0.2.0
  runnerVersions:
    - v0.2.0
    - v0.1.0
  fleetManagerVersion: v0.1.3

tail: keep
EOF
}

run_updater() {
  : >"${CURL_SEQUENCE}"
  node "${UPDATER}" "${TMP_DIR}/values.yaml" >"${TMP_DIR}/stdout" 2>"${TMP_DIR}/stderr"
}

assert_unchanged() {
  if ! cmp -s "${TMP_DIR}/values.before" "${TMP_DIR}/values.yaml"; then
    diff -u "${TMP_DIR}/values.before" "${TMP_DIR}/values.yaml" >&2 || true
    echo "lookup failure changed saved chart values" >&2
    exit 1
  fi
}

assert_file() {
  local expected="$1"
  if ! cmp -s "${expected}" "${TMP_DIR}/values.yaml"; then
    diff -u "${expected}" "${TMP_DIR}/values.yaml" >&2 || true
    echo "chart values do not match the selected releases" >&2
    exit 1
  fi
}

cat >"${TMP_DIR}/page-one.json" <<'EOF'
[
  {"tag_name": "v9.9.9", "draft": false, "prerelease": false},
  {"tag_name": "runner/v0.2.9", "draft": false, "prerelease": false},
  {"tag_name": "runner/v0.3.1-rc.1", "draft": false, "prerelease": true},
  {"tag_name": "runner/v0.4.0-rc.1", "draft": false, "prerelease": false},
  {"tag_name": "runner/v9.0.0", "draft": true, "prerelease": false},
  {"tag_name": "fleet-manager/v0.2.9", "draft": false, "prerelease": false},
  {"tag_name": "fleet-manager/v0.3.1-rc.1", "draft": false, "prerelease": true},
  {"tag_name": "fleet-manager/v0.4.0-rc.1", "draft": false, "prerelease": false},
  {"tag_name": "fleet-manager/v9.0.0", "draft": true, "prerelease": false}
]
EOF
cat >"${TMP_DIR}/page-two.json" <<'EOF'
[
  {"tag_name": "runner/v0.3.0", "draft": false, "prerelease": false},
  {"tag_name": "fleet-manager/v0.3.0", "draft": false, "prerelease": false}
]
EOF

write_saved_values
export CURL_MODE=success
run_updater
cat >"${TMP_DIR}/expected.yaml" <<EOF
other: keep

${FALLBACK_COMMENT}
componentReleases:
  runnerVersion: v0.3.0
  runnerVersions:
    - v0.3.0
  fleetManagerVersion: v0.3.0

tail: keep
EOF
assert_file "${TMP_DIR}/expected.yaml"
grep -q 'Runner default: v0.3.0' "${TMP_DIR}/stdout"
grep -q 'Fleet Manager default: v0.3.0' "${TMP_DIR}/stdout"
grep -Fxq "curl ${API_ROOT}" "${CURL_SEQUENCE}"
grep -Fxq "curl ${PAGE_TWO}" "${CURL_SEQUENCE}"
test "$(cat "${CURL_AUTH_FLAG}")" = "absent"

write_saved_values
export GITHUB_TOKEN="chart-release-test-token"
export CURL_MODE=success
run_updater
test "$(cat "${CURL_AUTH_FLAG}")" = "present"
unset GITHUB_TOKEN

write_saved_values
printf '%s\n' \
  '[{"tag_name":"runner/v0.9.0","draft":false,"prerelease":false},{"tag_name":"runner/v0.10.0","draft":false,"prerelease":false},{"tag_name":"fleet-manager/v0.9.0","draft":false,"prerelease":false},{"tag_name":"fleet-manager/v0.10.0","draft":false,"prerelease":false}]' \
  >"${TMP_DIR}/page-one.json"
export CURL_MODE=numeric
run_updater
cat >"${TMP_DIR}/expected.yaml" <<EOF
other: keep

${FALLBACK_COMMENT}
componentReleases:
  runnerVersion: v0.10.0
  runnerVersions:
    - v0.10.0
  fleetManagerVersion: v0.10.0

tail: keep
EOF
assert_file "${TMP_DIR}/expected.yaml"

write_saved_values
cp "${TMP_DIR}/values.yaml" "${TMP_DIR}/values.before"
export CURL_MODE=http-error
run_updater
assert_unchanged
grep -q 'HTTP 500' "${TMP_DIR}/stderr"

write_saved_values
cp "${TMP_DIR}/values.yaml" "${TMP_DIR}/values.before"
export CURL_MODE=stall
rm -f "${TMP_DIR}/max-time"
: >"${CURL_SEQUENCE}"
stall_started="$(date +%s)"
set +e
node "${UPDATER}" "${TMP_DIR}/values.yaml" >"${TMP_DIR}/stdout" 2>"${TMP_DIR}/stderr"
stall_status="$?"
set -e
stall_elapsed="$(( $(date +%s) - stall_started ))"
if [ "${stall_status}" -ne 0 ]; then
  echo "stalled release lookup did not fall back (status ${stall_status}, elapsed ${stall_elapsed}s)" >&2
  cat "${TMP_DIR}/stderr" >&2 || true
  exit 1
fi
if [ "${stall_elapsed}" -lt 10 ]; then
  echo "stalled release lookup returned before the request timeout (${stall_elapsed}s)" >&2
  exit 1
fi
assert_unchanged
grep -q 'request timed out' "${TMP_DIR}/stderr"
max_time="$(cat "${TMP_DIR}/max-time")"
if [ "${max_time}" -lt 10 ] || [ "${max_time}" -gt 30 ]; then
  echo "curl --max-time must stay between 10 and 30 seconds, got ${max_time}" >&2
  exit 1
fi

write_saved_values
cp "${TMP_DIR}/values.yaml" "${TMP_DIR}/values.before"
export CURL_MODE=invalid-json
run_updater
assert_unchanged
grep -q 'invalid JSON' "${TMP_DIR}/stderr"

write_saved_values
cp "${TMP_DIR}/values.yaml" "${TMP_DIR}/values.before"
printf '%s\n' \
  '[{"tag_name":"runner/v0.2.9","draft":false,"prerelease":false},{"tag_name":"fleet-manager/v0.2.9","draft":false,"prerelease":false}]' \
  >"${TMP_DIR}/page-one.json"
export CURL_MODE=page-two-error
run_updater
assert_unchanged
grep -q 'HTTP 502' "${TMP_DIR}/stderr"

write_saved_values
cp "${TMP_DIR}/values.yaml" "${TMP_DIR}/values.before"
export CURL_MODE=cycle
run_updater
assert_unchanged
grep -q 'pagination contains a cycle' "${TMP_DIR}/stderr"

write_saved_values
printf '%s\n' \
  '[{"tag_name":"runner/v0.3.0","draft":false,"prerelease":false},{"tag_name":"fleet-manager/v0.3.1-rc.1","draft":false,"prerelease":true}]' \
  >"${TMP_DIR}/page-one.json"
export CURL_MODE=fleet-manager-missing
run_updater
cat >"${TMP_DIR}/expected.yaml" <<EOF
other: keep

${FALLBACK_COMMENT}
componentReleases:
  runnerVersion: v0.3.0
  runnerVersions:
    - v0.3.0
  fleetManagerVersion: v0.1.3

tail: keep
EOF
assert_file "${TMP_DIR}/expected.yaml"
grep -q 'no stable Fleet Manager release' "${TMP_DIR}/stderr"

write_saved_values
printf '%s\n' \
  '[{"tag_name":"runner/v0.3.1-rc.1","draft":false,"prerelease":true},{"tag_name":"runner/v0.4.0-rc.1","draft":false,"prerelease":false},{"tag_name":"fleet-manager/v0.3.0","draft":false,"prerelease":false}]' \
  >"${TMP_DIR}/page-one.json"
export CURL_MODE=runner-missing
run_updater
cat >"${TMP_DIR}/expected.yaml" <<EOF
other: keep

${FALLBACK_COMMENT}
componentReleases:
  runnerVersion: v0.2.0
  runnerVersions:
    - v0.2.0
    - v0.1.0
  fleetManagerVersion: v0.3.0

tail: keep
EOF
assert_file "${TMP_DIR}/expected.yaml"
grep -q 'no stable runner release' "${TMP_DIR}/stderr"

: >"${CURL_SEQUENCE}"
export CURL_MODE=http-error
(
  cd "${REPO_ROOT}"
  bash "${BUILD_SCRIPT}" v1.2.3 "${TMP_DIR}/chart-output" >"${TMP_DIR}/build-stdout" 2>"${TMP_DIR}/build-stderr"
)
cmp -s "${VALUES_BACKUP}" "${VALUES_PATH}"
grep -q 'HTTP 500' "${TMP_DIR}/build-stderr"
curl_line="$(grep -n '^curl ' "${CURL_SEQUENCE}" | head -n 1 | cut -d: -f1)"
package_line="$(grep -n '^helm package$' "${CURL_SEQUENCE}" | head -n 1 | cut -d: -f1)"
test -n "${curl_line}"
test -n "${package_line}"
test "${curl_line}" -lt "${package_line}"
if grep -qx 'helm push' "${CURL_SEQUENCE}"; then
  echo "chart preparation must not publish to the registry" >&2
  exit 1
fi
test -f "${TMP_DIR}/chart-output/superplane-1.2.3.tgz"

: >"${CURL_SEQUENCE}"
export CURL_MODE=success
export COMPONENT_RELEASES_ALREADY_RESOLVED=yes
(
  cd "${REPO_ROOT}"
  bash "${BUILD_SCRIPT}" v1.2.3 "${TMP_DIR}/chart-output-reused" >"${TMP_DIR}/reuse-stdout" 2>"${TMP_DIR}/reuse-stderr"
)
unset COMPONENT_RELEASES_ALREADY_RESOLVED
if grep -q '^curl ' "${CURL_SEQUENCE}"; then
  echo "chart build looked up releases after the archive resolve" >&2
  exit 1
fi
cmp -s "${VALUES_BACKUP}" "${VALUES_PATH}"
test -f "${TMP_DIR}/chart-output-reused/superplane-1.2.3.tgz"

cat >"${TMP_DIR}/candidate-values.yaml" <<'EOF'
candidateOnly: keep

componentReleases:
  runnerVersion: v0.2.1
  runnerVersions:
    - v0.2.1
  fleetManagerVersion: v0.1.5

candidateTail: keep
EOF
cat >"${TMP_DIR}/stable-values.yaml" <<'EOF'
stableOnly: keep

componentReleases:
  runnerVersion: v9.0.0
  runnerVersions:
    - v9.0.0
  fleetManagerVersion: v9.0.0

stableTail: keep
EOF
node "${UPDATER}" reuse \
  "${TMP_DIR}/candidate-values.yaml" \
  "${TMP_DIR}/stable-values.yaml"
cat >"${TMP_DIR}/expected-reused-values.yaml" <<'EOF'
stableOnly: keep

componentReleases:
  runnerVersion: v0.2.1
  runnerVersions:
    - v0.2.1
  fleetManagerVersion: v0.1.5

stableTail: keep
EOF
if ! cmp -s "${TMP_DIR}/expected-reused-values.yaml" "${TMP_DIR}/stable-values.yaml"; then
  diff -u "${TMP_DIR}/expected-reused-values.yaml" "${TMP_DIR}/stable-values.yaml" >&2 || true
  echo "candidate component release defaults were not reused" >&2
  exit 1
fi

echo "Chart component release default tests passed."
