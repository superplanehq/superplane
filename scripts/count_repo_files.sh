#!/usr/bin/env bash
#
# Prints the total number of files tracked in the repository.

set -euo pipefail

cd "$(dirname "$0")/.."

count="$(git ls-files | wc -l | tr -d '[:space:]')"

echo "Total files in repo: $count"
