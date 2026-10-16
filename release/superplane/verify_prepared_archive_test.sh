#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

ARCHIVE="${TMP_DIR}/superplane-amd64.oci.tar"
printf '{"schemaVersion":2}\n' >"${TMP_DIR}/index.json"
tar -C "${TMP_DIR}" -cf "${ARCHIVE}" index.json
sha256sum "${ARCHIVE}" | awk '{print $1}' >"${ARCHIVE}.sha256"
cp "${TMP_DIR}/index.json" "${ARCHIVE}.descriptor.json"

bash "${SCRIPT_DIR}/verify-prepared-archive.sh" "${ARCHIVE}" >/dev/null

printf '0\n' >"${ARCHIVE}.sha256"
if bash "${SCRIPT_DIR}/verify-prepared-archive.sh" "${ARCHIVE}" >/dev/null 2>&1; then
  echo "A changed OCI archive must fail checksum verification" >&2
  exit 1
fi

sha256sum "${ARCHIVE}" | awk '{print $1}' >"${ARCHIVE}.sha256"
printf '{}\n' >"${ARCHIVE}.descriptor.json"
if bash "${SCRIPT_DIR}/verify-prepared-archive.sh" "${ARCHIVE}" >/dev/null 2>&1; then
  echo "A changed OCI descriptor must fail verification" >&2
  exit 1
fi

echo "SuperPlane prepared archive tests passed."
