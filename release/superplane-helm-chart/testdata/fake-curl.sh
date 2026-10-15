#!/bin/sh

set -eu

output_file=""
has_body=no
arguments="$*"

while [ "$#" -gt 0 ]; do
  case "$1" in
    --output)
      output_file="$2"
      shift 2
      ;;
    --data-binary)
      has_body=yes
      shift 2
      ;;
    *)
      shift
      ;;
  esac
done

printf '%s\n' "${arguments}" >>"${TEST_ARGS_FILE}"
if [ "${has_body}" = "yes" ]; then
  cat >"${TEST_BODY_FILE}"
fi

call_count="$(cat "${TEST_STATE_FILE}")"
call_count=$((call_count + 1))
printf '%s' "${call_count}" >"${TEST_STATE_FILE}"
status="$(printf '%s' "${TEST_SEQUENCE}" | cut -d, -f"${call_count}")"

if [ "${status}" = "200" ]; then
  printf '%s' '{"data":{"INSTALLATION_ADMIN_TOKEN":"ZXhpc3Rpbmc="}}' >"${output_file}"
else
  printf '%s' '{}' >"${output_file}"
fi

printf '%s' "${status}"
