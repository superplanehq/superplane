#!/usr/bin/env bash

# Pin yt-dlp next to the media toolchain. Tasks must not download it.
set -euo pipefail

YT_DLP_VERSION="${YT_DLP_VERSION:-2026.08.19}"
YT_DLP_SHA256="${YT_DLP_SHA256:-1fa6733c37ea6fb51c99ad8fe785e7b7e5f3246c9b980230329d4fb72ed8d4d6}"
YT_DLP_URL="${YT_DLP_URL:-https://github.com/yt-dlp/yt-dlp/releases/download/${YT_DLP_VERSION}/yt-dlp}"
PREFIX="${PREFIX:-/usr/local}"

if ! command -v python3 >/dev/null 2>&1; then
  printf 'python3 must be installed before yt-dlp.\n' >&2
  exit 1
fi

tmp="$(mktemp)"
trap 'rm -f "${tmp}"' EXIT
curl --fail --location --silent --show-error "${YT_DLP_URL}" --output "${tmp}"
echo "${YT_DLP_SHA256}  ${tmp}" | sha256sum -c -
install -D -m 0755 "${tmp}" "${PREFIX}/bin/yt-dlp"
yt-dlp --version
