#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

if [ "${1-}" = "" ] || [ "${2-}" = "" ]; then
  echo "Usage: release/superplane-demo-image/download-supergit.sh <version> <arch>"
  echo ""
  echo "Example:"
  echo "  release/superplane-demo-image/download-supergit.sh v0.1.1 amd64"
  exit 1
fi

VERSION="$1"
ARCH="$2"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

case "${ARCH}" in
  amd64 | arm64) ;;
  *)
    echo "Unsupported architecture: ${ARCH} (expected amd64 or arm64)" >&2
    exit 1
    ;;
esac

VERSION="${VERSION#v}"
TAG="v${VERSION}"
ASSET="supergit-linux-${ARCH}"
BASE_URL="https://github.com/superplanehq/supergit/releases/download/${TAG}"
OUT_DIR="${REPO_ROOT}/build/superplane-demo-supergit"
OUT_FILE="${OUT_DIR}/supergit"

mkdir -p "${OUT_DIR}"

TMP_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

echo "* Downloading SuperGit ${TAG} (${ASSET})"
curl -fsSL "${BASE_URL}/checksums.txt" -o "${TMP_DIR}/checksums.txt"
curl -fsSL "${BASE_URL}/${ASSET}" -o "${TMP_DIR}/${ASSET}"

EXPECTED="$(grep " ${ASSET}$" "${TMP_DIR}/checksums.txt" | awk '{print $1}')"
if [ -z "${EXPECTED}" ]; then
  echo "Checksum not found for ${ASSET} in release ${TAG}" >&2
  exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
  ACTUAL="$(sha256sum "${TMP_DIR}/${ASSET}" | awk '{print $1}')"
elif command -v shasum >/dev/null 2>&1; then
  ACTUAL="$(shasum -a 256 "${TMP_DIR}/${ASSET}" | awk '{print $1}')"
else
  echo "sha256sum or shasum is required to verify SuperGit downloads" >&2
  exit 1
fi

if [ "${EXPECTED}" != "${ACTUAL}" ]; then
  echo "Checksum mismatch for ${ASSET}" >&2
  echo "  expected: ${EXPECTED}" >&2
  echo "  actual:   ${ACTUAL}" >&2
  exit 1
fi

mv "${TMP_DIR}/${ASSET}" "${OUT_FILE}"
chmod +x "${OUT_FILE}"

echo "* SuperGit binary ready at ${OUT_FILE}"
