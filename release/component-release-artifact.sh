#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

if [ "$#" -ne 4 ]; then
  echo "Usage: release/component-release-artifact.sh <push|pull> <component> <version> <release-directory>" >&2
  exit 1
fi

ACTION="$1"
COMPONENT="$2"
VERSION="$3"
RELEASE_DIR="$4"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

case "${ACTION}" in
  push | pull) ;;
  *)
    echo "Error: action must be push or pull" >&2
    exit 1
    ;;
esac

case "${COMPONENT}" in
  runner | fleet-manager) ;;
  *)
    echo "Error: unsupported component ${COMPONENT}" >&2
    exit 1
    ;;
esac

node "${SCRIPT_DIR}/component-release.js" validate-version "${VERSION}" >/dev/null

if [ "$(basename "${RELEASE_DIR}")" != "${VERSION}" ] || \
  [ "$(basename "$(dirname "${RELEASE_DIR}")")" != "${COMPONENT}" ]; then
  echo "Error: release directory must end with ${COMPONENT}/${VERSION}" >&2
  exit 1
fi

BUNDLE_NAME="${COMPONENT}-${VERSION}.tar.gz"
BUNDLE_PATH="$(pwd)/${BUNDLE_NAME}"
RELEASE_PARENT="$(dirname "${RELEASE_DIR}")"

push_bundle() {
  if [ ! -d "${RELEASE_DIR}" ]; then
    echo "Error: release directory ${RELEASE_DIR} does not exist" >&2
    exit 1
  fi
  if [ -e "${BUNDLE_PATH}" ]; then
    echo "Error: release bundle ${BUNDLE_NAME} already exists" >&2
    exit 1
  fi

  tar -C "${RELEASE_PARENT}" -czf "${BUNDLE_PATH}" "${VERSION}"
  artifact push workflow "${BUNDLE_NAME}"
}

pull_bundle() {
  if [ -e "${RELEASE_DIR}" ]; then
    echo "Error: release directory ${RELEASE_DIR} already exists" >&2
    exit 1
  fi

  artifact pull workflow "${BUNDLE_NAME}"

  listing_dir="$(mktemp -d)"
  trap 'rm -rf "${listing_dir}"' EXIT

  if ! tar -tzf "${BUNDLE_PATH}" >"${listing_dir}/paths"; then
    echo "Error: release bundle is not a valid gzip archive" >&2
    exit 1
  fi
  if ! tar -tvzf "${BUNDLE_PATH}" >"${listing_dir}/entries"; then
    echo "Error: release bundle metadata cannot be read" >&2
    exit 1
  fi

  while IFS= read -r entry; do
    case "${entry}" in
      "${VERSION}" | "${VERSION}/" | "${VERSION}/"*)
        if [[ "/${entry}/" == *"/../"* ]]; then
          echo "Error: release bundle contains unsafe path ${entry}" >&2
          exit 1
        fi
        ;;
      *)
        echo "Error: release bundle contains unexpected path ${entry}" >&2
        exit 1
        ;;
    esac
  done <"${listing_dir}/paths"

  while IFS= read -r entry; do
    case "${entry:0:1}" in
      - | d) ;;
      *)
        echo "Error: release bundle contains a link or special file" >&2
        exit 1
        ;;
    esac
  done <"${listing_dir}/entries"

  mkdir -p "${RELEASE_PARENT}"
  tar -C "${RELEASE_PARENT}" -xzf "${BUNDLE_PATH}"
}

if [ "${ACTION}" = "push" ]; then
  push_bundle
else
  pull_bundle
fi
