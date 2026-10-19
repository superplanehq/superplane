#!/usr/bin/env bash

set -euo pipefail

if [ "$#" -ne 4 ]; then
  echo "Usage: release/cli/build.sh <version> <os> <architecture> <output>" >&2
  exit 1
fi

VERSION="$1"
OS="$2"
ARCHITECTURE="$3"
OUTPUT="$4"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

node "${REPO_ROOT}/release/component-release.js" validate-version "${VERSION}" >/dev/null
case "${OS}/${ARCHITECTURE}" in
  darwin/amd64 | darwin/arm64 | linux/amd64 | linux/arm64) ;;
  *)
    echo "Error: unsupported CLI platform ${OS}/${ARCHITECTURE}" >&2
    exit 1
    ;;
esac

mkdir -p "$(dirname "${OUTPUT}")"
cd "${REPO_ROOT}"
make cli.build.binary \
  OS="${OS}" \
  ARCH="${ARCHITECTURE}" \
  CLI_VERSION="${VERSION}" \
  CLI_OUTPUT="${OUTPUT}"
