#!/usr/bin/env bash
set -euo pipefail

# Run Bun UI unit tests with compact dots output and a JUnit report
# at the repo root (same path Semaphore publishes for Go tests).
#
# Usage:
#   bash scripts/test_ui_autoparallel.sh
#   FILES="src/lib/duration.spec.ts" bash scripts/test_ui_autoparallel.sh
#   SHARD_INDEX=1 SHARD_COUNT=4 bash scripts/test_ui_autoparallel.sh

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${script_dir}/../web_src"

junit_file="${JUNIT_FILE:-/app/junit-report.xml}"
args=(
  --dots
  --reporter=junit
  --reporter-outfile="${junit_file}"
)

if [[ -n "${SHARD_COUNT:-}" ]]; then
  source "${script_dir}/lib/shard_args.sh"
  echo "Running UI unit tests shard ${SHARD_INDEX}/${SHARD_COUNT}"
  args+=(--parallel --shard="${SHARD_INDEX}/${SHARD_COUNT}")
else
  args+=(--isolate)
fi

if [[ -n "${FILES:-}" ]]; then
  read -r -a file_args <<< "${FILES}"
  bun test "${args[@]}" "${file_args[@]}"
else
  bun test "${args[@]}"
fi
