#!/bin/sh

set -eu

: "${SECRET_NAME:?SECRET_NAME is required}"
: "${TOKEN_KEY:?TOKEN_KEY is required}"
: "${POD_NAMESPACE:?POD_NAMESPACE is required}"

umask 077
work_dir="$(mktemp -d)"
trap 'rm -rf "$work_dir"' EXIT HUP INT TERM

api_url="https://${KUBERNETES_SERVICE_HOST}:${KUBERNETES_SERVICE_PORT_HTTPS}/api/v1/namespaces/${POD_NAMESPACE}/secrets/${SECRET_NAME}"
service_account_dir="${SERVICE_ACCOUNT_DIR:-/var/run/secrets/kubernetes.io/serviceaccount}"
curl_config="${work_dir}/curl.conf"
response_file="${work_dir}/response.json"

printf 'header = "Authorization: Bearer %s"\n' "$(cat "${service_account_dir}/token")" >"${curl_config}"
printf 'cacert = "%s/ca.crt"\n' "${service_account_dir}" >>"${curl_config}"
printf '%s\n' \
  'silent' \
  'show-error' \
  'connect-timeout = 10' \
  'max-time = 30' \
  'retry = 5' \
  'retry-delay = 2' \
  'retry-all-errors' >>"${curl_config}"

request() {
  method="$1"
  output="$2"
  shift 2

  curl --config "${curl_config}" \
    --request "${method}" \
    --output "${output}" \
    --write-out '%{http_code}' \
    "$@" \
    "${api_url}"
}

validate_existing_secret() {
  status="$(request GET "${response_file}")"
  [ "${status}" = "200" ] || return 1
  grep -Fq "\"${TOKEN_KEY}\":" "${response_file}"
}

status="$(request GET "${response_file}")"
case "${status}" in
  200)
    if ! grep -Fq "\"${TOKEN_KEY}\":" "${response_file}"; then
      echo "Fleet Manager credential Secret does not contain the configured key" >&2
      exit 1
    fi
    exit 0
    ;;
  404)
    ;;
  *)
    echo "Could not read the Fleet Manager credential Secret (HTTP ${status})" >&2
    exit 1
    ;;
esac

token="$(od -An -N32 -tx1 /dev/urandom | tr -d ' \n')"
encoded_token="$(printf '%s' "${token}" | base64 | tr -d '\n')"
payload="$(printf '{\"apiVersion\":\"v1\",\"kind\":\"Secret\",\"metadata\":{\"name\":\"%s\",\"namespace\":\"%s\"},\"type\":\"Opaque\",\"data\":{\"%s\":\"%s\"}}' \
  "${SECRET_NAME}" "${POD_NAMESPACE}" "${TOKEN_KEY}" "${encoded_token}")"
unset token encoded_token

status="$(printf '%s' "${payload}" | request POST "${response_file}" \
  --header 'Content-Type: application/json' \
  --data-binary @-)"
unset payload

case "${status}" in
  201)
    exit 0
    ;;
  409)
    if validate_existing_secret; then
      exit 0
    fi
    echo "A concurrent Secret creation did not produce the configured key" >&2
    exit 1
    ;;
  *)
    echo "Could not create the Fleet Manager credential Secret (HTTP ${status})" >&2
    exit 1
    ;;
esac
