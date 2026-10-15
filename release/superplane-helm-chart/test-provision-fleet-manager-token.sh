#!/usr/bin/env bash

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SCRIPT="${REPO_ROOT}/release/superplane-helm-chart/helm/files/provision-fleet-manager-token.sh"
FAKE_CURL="${REPO_ROOT}/release/superplane-helm-chart/testdata/fake-curl.sh"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

mkdir -p "${TMP_DIR}/bin" "${TMP_DIR}/service-account"
cp "${FAKE_CURL}" "${TMP_DIR}/bin/curl"
chmod +x "${TMP_DIR}/bin/curl"
printf '%s' 'kubernetes-api-token' >"${TMP_DIR}/service-account/token"
printf '%s' 'test-ca' >"${TMP_DIR}/service-account/ca.crt"

run_case() {
  name="$1"
  sequence="$2"
  expected_calls="$3"
  expect_success="$4"
  case_dir="${TMP_DIR}/${name}"
  mkdir -p "${case_dir}"
  printf '0' >"${case_dir}/state"

  set +e
  PATH="${TMP_DIR}/bin:${PATH}" \
    TEST_SEQUENCE="${sequence}" \
    TEST_STATE_FILE="${case_dir}/state" \
    TEST_BODY_FILE="${case_dir}/body" \
    TEST_ARGS_FILE="${case_dir}/args" \
    SERVICE_ACCOUNT_DIR="${TMP_DIR}/service-account" \
    KUBERNETES_SERVICE_HOST=kubernetes.default.svc \
    KUBERNETES_SERVICE_PORT_HTTPS=443 \
    SECRET_NAME=superplane-fleet-manager-token \
    TOKEN_KEY=INSTALLATION_ADMIN_TOKEN \
    POD_NAMESPACE=superplane \
    sh "${SCRIPT}" >"${case_dir}/stdout" 2>"${case_dir}/stderr"
  result=$?
  set -e

  if [ "${expect_success}" = "yes" ] && [ "${result}" -ne 0 ]; then
    cat "${case_dir}/stderr" >&2
    echo "${name}: expected success" >&2
    exit 1
  fi
  if [ "${expect_success}" = "no" ] && [ "${result}" -eq 0 ]; then
    echo "${name}: expected failure" >&2
    exit 1
  fi

  actual_calls="$(cat "${case_dir}/state")"
  if [ "${actual_calls}" != "${expected_calls}" ]; then
    echo "${name}: got ${actual_calls} calls, expected ${expected_calls}" >&2
    exit 1
  fi

  if grep -q 'kubernetes-api-token' "${case_dir}/args"; then
    echo "${name}: Kubernetes token entered process arguments" >&2
    exit 1
  fi
  if grep -q 'kubernetes-api-token' "${case_dir}/stdout" "${case_dir}/stderr"; then
    echo "${name}: Kubernetes token entered output" >&2
    exit 1
  fi
}

run_case preserves-existing 200 1 yes
run_case creates-missing 404,201 2 yes
grep -q '"INSTALLATION_ADMIN_TOKEN":"' "${TMP_DIR}/creates-missing/body"
run_case accepts-concurrent-create 404,409,200 3 yes
run_case rejects-api-error 403 1 no
grep -q 'HTTP 403' "${TMP_DIR}/rejects-api-error/stderr"
