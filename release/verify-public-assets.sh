#!/usr/bin/env bash

set -euo pipefail

if [ "$#" -lt 3 ]; then
  echo "Usage: release/verify-public-assets.sh <base-url> <directory> <asset>..." >&2
  exit 1
fi

BASE_URL="${1%/}"
DIRECTORY="$2"
shift 2

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

for asset in "$@"; do
  local_checksum="$(sha256_file "${DIRECTORY}/${asset}")"
  remote_checksum="$(curl --fail --location --silent --show-error \
    "${BASE_URL}/${asset}" | sha256sum | awk '{print $1}')"
  if [ "${remote_checksum}" != "${local_checksum}" ]; then
    echo "Error: public asset ${BASE_URL}/${asset} does not match ${DIRECTORY}/${asset}" >&2
    exit 1
  fi
  echo "* Verified ${BASE_URL}/${asset}"
done
