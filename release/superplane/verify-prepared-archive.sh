#!/usr/bin/env bash

set -euo pipefail

if [ "$#" -ne 1 ]; then
  echo "Usage: release/superplane/verify-prepared-archive.sh <oci-archive>" >&2
  exit 1
fi

ARCHIVE="$1"
CHECKSUM_FILE="${ARCHIVE}.sha256"
DESCRIPTOR_FILE="${ARCHIVE}.descriptor.json"

if [ ! -f "${ARCHIVE}" ] || [ ! -f "${CHECKSUM_FILE}" ] || [ ! -f "${DESCRIPTOR_FILE}" ]; then
  echo "Error: ${ARCHIVE} requires its checksum and descriptor artifacts" >&2
  exit 1
fi

expected="$(tr -d '[:space:]' <"${CHECKSUM_FILE}")"
actual="$(sha256sum "${ARCHIVE}" | awk '{print $1}')"
if [ "${expected}" != "${actual}" ]; then
  echo "Error: ${ARCHIVE} checksum does not match ${CHECKSUM_FILE}" >&2
  exit 1
fi

if ! cmp -s "${DESCRIPTOR_FILE}" <(tar -xOf "${ARCHIVE}" index.json); then
  echo "Error: ${ARCHIVE} descriptor does not match index.json" >&2
  exit 1
fi

echo "Prepared OCI archive ${ARCHIVE} matches its descriptor and checksum."
