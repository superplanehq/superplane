#!/usr/bin/env bash
#
# Store the Go module cache (tmp/go) and build cache (tmp/go-build) on CI.
#
# Only the default branch uploads. The keys hold a go.sum checksum and no branch
# name, so a feature branch would populate the entry that every other branch
# reads, and each upload adds gigabytes to the project cache quota. Feature
# branches restore the entry from the default branch and download only the
# modules that their own go.sum adds.
#
# The build key also includes the Go version from go.mod. Existing cache entries
# cannot be replaced, so a Go update must use a new key to store compiled packages
# for that version. The entry keeps the compiled standard library and
# dependencies. Packages of this repository change
# with every commit and miss the build cache in any case.

set -euo pipefail

DEFAULT_BRANCH="main"

if [[ "${SEMAPHORE_GIT_BRANCH:-}" != "$DEFAULT_BRANCH" ]]; then
  echo "Branch is not ${DEFAULT_BRANCH}; keep the Go caches as they are."
  exit 0
fi

go_sum_checksum="$(checksum go.sum)"
go_version="$(awk '/^go / {print $2}' go.mod)"

cache store "go-mod-${go_sum_checksum}" tmp/go
cache store "go-build-${go_version}-${go_sum_checksum}" tmp/go-build
