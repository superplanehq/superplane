#!/usr/bin/env bash
set -euo pipefail

# Run Bun UI unit tests with compact dots output and a JUnit report
# at the repo root (same path Semaphore publishes for Go tests).
#
# Usage:
#   bash scripts/test_ui_autoparallel.sh
#   FILES="src/lib/duration.spec.ts" bash scripts/test_ui_autoparallel.sh
#   SHARD_INDEX=1 SHARD_COUNT=2 bash scripts/test_ui_autoparallel.sh

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"
cd "${repo_root}/web_src"

junit_file="${JUNIT_FILE:-${repo_root}/junit-report.xml}"
args=(
  --dots
  --reporter=junit
  --reporter-outfile="${junit_file}"
)

if [[ -n "${SHARD_COUNT:-}" ]]; then
  source "${script_dir}/lib/shard_args.sh"
  echo "Running UI unit tests shard ${SHARD_INDEX}/${SHARD_COUNT}"
  args+=(--parallel=1 --shard="${SHARD_INDEX}/${SHARD_COUNT}")
else
  args+=(--isolate)
fi

file_args=()
if [[ -n "${FILES:-}" ]]; then
  read -r -a file_args <<< "${FILES}"
fi

run_bun_test() {
  bun test "${args[@]}" "${file_args[@]}"
}

bun_status=0
log_file="$(mktemp)"
run_bun_test 2>&1 | tee "${log_file}" || bun_status=$?
if [[ "${bun_status}" -ne 0 ]] && grep -q "worker crashed" "${log_file}"; then
  echo "UI test worker crashed. Retry the run once."
  bun_status=0
  run_bun_test || bun_status=$?
fi
rm -f "${log_file}"

bun "${script_dir}/flatten_junit.mjs" "${junit_file}"
exit "${bun_status}"
