#!/usr/bin/env bash
# Exercises scripts/count_repo_files.sh against the real tree.

set -euo pipefail

cd "$(dirname "$0")/.."

out="$(bash ./scripts/count_repo_files.sh)"
echo "$out"

if [[ ! "$out" =~ ^Total\ files\ in\ repo:\ [0-9]+$ ]]; then
	echo "FAIL: unexpected output format: $out" >&2
	exit 1
fi

count="${out##*: }"
tracked="$(git ls-files | wc -l | tr -d '[:space:]')"

if [ "$count" != "$tracked" ]; then
	echo "FAIL: reported count ($count) does not match git ls-files count ($tracked)" >&2
	exit 1
fi

echo "PASS count_repo_files"
