#!/usr/bin/env bash
set -euo pipefail

# Shard Go unit tests across CI workers.
# Usage (Semaphore example):
#   make test.coverage.autoparallel SHARD_INDEX=$SEMAPHORE_JOB_INDEX SHARD_COUNT=$SEMAPHORE_JOB_COUNT

source "$(dirname "${BASH_SOURCE[0]}")/lib/shard_args.sh"
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
timings_file="${TEST_TIMINGS_FILE:-${script_dir}/test_timings.json}"

echo "Running unit tests shard ${SHARD_INDEX}/${SHARD_COUNT}"

module_prefix="$(go list -m)"

# Test file counts provide fallback weights for packages without timings.
weighted_packages=()
while IFS= read -r import_path; do
  package_dir="./${import_path#"$module_prefix"/}"
  test_file_count="$(find "$package_dir" -maxdepth 1 -type f -name '*_test.go' | wc -l | tr -d ' ')"
  weighted_packages+=("${package_dir}"$'\t'"${test_file_count}")
done < <(go list ./pkg/... ./ee/...)

if [[ "${#weighted_packages[@]}" -eq 0 ]]; then
  echo "No packages found under ./pkg or ./ee, nothing to run."
  exit 0
fi

# Capture the helper status before reading its output, so errors cannot silently
# turn a shard into an empty successful job.
assignment="$(printf '%s\n' "${weighted_packages[@]}" |
  python3 "${script_dir}/lib/test_shards.py" unit "$SHARD_INDEX" "$SHARD_COUNT" "$timings_file")"
shard_packages=()
if [[ -n "$assignment" ]]; then
  mapfile -t shard_packages <<< "$assignment"
fi

if [[ "${#shard_packages[@]}" -eq 0 ]]; then
  echo "No packages assigned to shard ${SHARD_INDEX}/${SHARD_COUNT}; exiting successfully."
  exit 0
fi

echo "Selected packages for shard ${SHARD_INDEX}/${SHARD_COUNT}:"
for package_dir in "${shard_packages[@]}"; do
  echo "  - ${package_dir}"
done
echo ""

# gotestsum expects the package patterns separated by spaces.
gotestsum \
  --format short \
  --junitfile junit-report.xml \
  --packages="${shard_packages[*]}" \
  -- \
  -p 1 \
  -coverprofile=coverage-go.out \
  -covermode=atomic

go tool cover -func=coverage-go.out | grep '^total:'

coverage_args=(--profile coverage-go.out)
if [[ "$SHARD_COUNT" -gt 1 ]]; then
  coverage_args+=(--ignore-missing-packages)
fi

go run ./scripts/check_go_coverage_budget.go "${coverage_args[@]}"
