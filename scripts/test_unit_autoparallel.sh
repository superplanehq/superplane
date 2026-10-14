#!/usr/bin/env bash
set -euo pipefail

# Shard Go unit tests across CI workers.
# Usage (Semaphore example):
#   make test.coverage.autoparallel SHARD_INDEX=$SEMAPHORE_JOB_INDEX SHARD_COUNT=$SEMAPHORE_JOB_COUNT

source "$(dirname "${BASH_SOURCE[0]}")/lib/shard_args.sh"
source "$(dirname "${BASH_SOURCE[0]}")/lib/weighted_shards.sh"

echo "Running unit tests shard ${SHARD_INDEX}/${SHARD_COUNT}"

module_prefix="$(go list -m)"

# Test file count estimates compile cost. Add measured test time for packages
# whose runtime is large enough to dominate a shard.
load_shard_weights "$(dirname "${BASH_SOURCE[0]}")/ci_unit_package_weights.tsv"

packages="$(go list ./pkg/... ./ee/... ./cmd/... ./test/support/...)"
if [[ -z "$packages" ]]; then
  echo "No unit test packages found." >&2
  exit 1
fi

weighted_packages=()
while IFS= read -r import_path; do
  package_dir="./${import_path#"$module_prefix"/}"
  test_file_count="$(find "$package_dir" -maxdepth 1 -type f -name '*_test.go' | wc -l | tr -d ' ')"
  package_path="${package_dir#./}"
  extra_weight="$(shard_weight_for "$package_path" 0)"
  # A package with no tests still has to compile in the selected job.
  weight=$(( (test_file_count > 0 ? test_file_count : 1) + extra_weight ))
  weighted_packages+=("${weight}"$'\t'"${package_dir}")
done <<< "$packages"

shard_packages=()
while IFS=$'\t' read -r shard package_dir; do
  if [[ "$shard" -eq "$SHARD_INDEX" ]]; then
    shard_packages+=("$package_dir")
  fi
done < <(printf '%s\n' "${weighted_packages[@]}" | assign_weighted_shards "$SHARD_COUNT")

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
