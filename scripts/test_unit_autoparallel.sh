#!/usr/bin/env bash
set -euo pipefail

# Shard Go unit tests across CI workers.
# Usage (Semaphore example):
#   make test.coverage.autoparallel SHARD_INDEX=$SEMAPHORE_JOB_INDEX SHARD_COUNT=$SEMAPHORE_JOB_COUNT

source "$(dirname "${BASH_SOURCE[0]}")/lib/shard_args.sh"

echo "Running unit tests shard ${SHARD_INDEX}/${SHARD_COUNT}"

module_prefix="$(go list -m)"

# Weigh each package by its test file count, so that heavy packages do not
# cluster on one shard.
weighted_packages=()
while IFS= read -r import_path; do
  package_dir="./${import_path#"$module_prefix"/}"
  test_file_count="$(find "$package_dir" -maxdepth 1 -type f -name '*_test.go' | wc -l | tr -d ' ')"
  weighted_packages+=("${test_file_count}:${package_dir}")
done < <(go list ./pkg/...)

if [[ "${#weighted_packages[@]}" -eq 0 ]]; then
  echo "No packages found under ./pkg, nothing to run."
  exit 0
fi

# Heaviest package first, so the greedy assignment below stays balanced.
sorted_packages=()
while IFS= read -r entry; do
  sorted_packages+=("$entry")
done < <(printf '%s\n' "${weighted_packages[@]}" | sort -t: -k1,1nr -k2,2)

shard_loads=()
for ((shard = 0; shard < SHARD_COUNT; shard++)); do
  shard_loads+=("0")
done

# Assign every package to the least loaded shard, and keep the ones for us.
shard_packages=()
for entry in "${sorted_packages[@]}"; do
  test_file_count="${entry%%:*}"
  package_dir="${entry#*:}"

  lightest_shard=0
  for ((shard = 1; shard < SHARD_COUNT; shard++)); do
    if [[ "${shard_loads[$shard]}" -lt "${shard_loads[$lightest_shard]}" ]]; then
      lightest_shard="$shard"
    fi
  done

  shard_loads[$lightest_shard]=$((shard_loads[lightest_shard] + test_file_count))
  if [[ $((lightest_shard + 1)) -eq "$SHARD_INDEX" ]]; then
    shard_packages+=("$package_dir")
  fi
done

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
