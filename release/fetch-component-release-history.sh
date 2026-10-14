#!/usr/bin/env bash

set -euo pipefail

if [ "$(git rev-parse --is-shallow-repository)" = true ]; then
  git fetch --unshallow --tags origin
  exit 0
fi

git fetch --tags origin
