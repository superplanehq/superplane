#!/usr/bin/env bash

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEST_DIRECTORY="$(mktemp -d)"
trap 'rm -rf "${TEST_DIRECTORY}"' EXIT
export GIT_CONFIG_GLOBAL=/dev/null
export GIT_CONFIG_NOSYSTEM=1

SOURCE_REPOSITORY="${TEST_DIRECTORY}/source"
REMOTE_REPOSITORY="${TEST_DIRECTORY}/remote.git"
SHALLOW_REPOSITORY="${TEST_DIRECTORY}/shallow"

git init --quiet "${SOURCE_REPOSITORY}"
git -C "${SOURCE_REPOSITORY}" config user.name "Release Test"
git -C "${SOURCE_REPOSITORY}" config user.email "release-test@example.com"
printf 'initial\n' >"${SOURCE_REPOSITORY}/release.txt"
git -C "${SOURCE_REPOSITORY}" add release.txt
git -C "${SOURCE_REPOSITORY}" commit --quiet -m "Initial release"
git -C "${SOURCE_REPOSITORY}" tag --no-sign v0.30.0
printf 'component change\n' >>"${SOURCE_REPOSITORY}/release.txt"
git -C "${SOURCE_REPOSITORY}" commit --quiet -am "Add component change"

git clone --quiet --bare "${SOURCE_REPOSITORY}" "${REMOTE_REPOSITORY}"
git clone --quiet --depth 1 "file://${REMOTE_REPOSITORY}" "${SHALLOW_REPOSITORY}"

test "$(git -C "${SHALLOW_REPOSITORY}" rev-parse --is-shallow-repository)" = true
if git -C "${SHALLOW_REPOSITORY}" merge-base --is-ancestor v0.30.0 HEAD 2>/dev/null; then
  echo "Expected the shallow checkout to omit the initial release tag" >&2
  exit 1
fi

(
  cd "${SHALLOW_REPOSITORY}"
  bash "${REPO_ROOT}/release/fetch-component-release-history.sh"
)

test "$(git -C "${SHALLOW_REPOSITORY}" rev-parse --is-shallow-repository)" = false
git -C "${SHALLOW_REPOSITORY}" merge-base --is-ancestor v0.30.0 HEAD

echo "Component release history tests passed."
