#!/usr/bin/env bash
set -euo pipefail

# Shard e2e Go tests across CI workers.
# Usage (Semaphore example):
#   make test.e2e.autoparallel SHARD_INDEX=$SEMAPHORE_JOB_INDEX SHARD_COUNT=$SEMAPHORE_JOB_COUNT

source "$(dirname "${BASH_SOURCE[0]}")/lib/shard_args.sh"
source "$(dirname "${BASH_SOURCE[0]}")/lib/weighted_shards.sh"

echo "Running e2e tests shard ${SHARD_INDEX}/${SHARD_COUNT}"

if [[ ! -d "./test/e2e" ]]; then
  echo "No ./test/e2e directory found, nothing to run."
  exit 0
fi

# Collect all top-level Test* functions from e2e test files.
all_tests=()
while IFS= read -r file; do
  while IFS= read -r name; do
    [[ -n "$name" && "$name" != "TestMain" ]] && all_tests+=("$name")
  done < <(awk '
    /^func[[:space:]]+Test[[:alnum:]_]*[[:space:]]*\(/ {
      line=$0
      sub(/^func[[:space:]]+/, "", line)
      sub(/\(.*/, "", line)
      gsub(/[[:space:]]+/, "", line)
      print line
    }
  ' "$file")
done < <(find ./test/e2e -maxdepth 1 -type f -name '*_test.go' | sort)

if [[ "${#all_tests[@]}" -eq 0 ]]; then
  echo "No e2e tests found in ./test/e2e, nothing to run."
  exit 0
fi

# Deduplicate and sort test names.
sorted_tests=()
while IFS= read -r test_name; do
  sorted_tests+=("$test_name")
done < <(printf '%s\n' "${all_tests[@]}" | LC_ALL=C sort -u)
all_tests=("${sorted_tests[@]}")

# Use recent root-test durations so long browser scenarios do not cluster on
# the same worker. A new test receives the median estimate until measured.
load_shard_weights "$(dirname "${BASH_SOURCE[0]}")/ci_e2e_test_weights.tsv"

weighted_tests=()
for test_name in "${all_tests[@]}"; do
  weight="$(shard_weight_for "$test_name" 30)"
  weighted_tests+=("${weight}"$'\t'"${test_name}")
done

selected_tests=()
while IFS=$'\t' read -r shard test_name; do
  if [[ "$shard" -eq "$SHARD_INDEX" ]]; then
    selected_tests+=("$test_name")
  fi
done < <(printf '%s\n' "${weighted_tests[@]}" | assign_weighted_shards "$SHARD_COUNT")

if [[ "${#selected_tests[@]}" -eq 0 ]]; then
  echo "No tests assigned to shard ${SHARD_INDEX}/${SHARD_COUNT}; exiting successfully."
  exit 0
fi

echo "Selected tests for shard ${SHARD_INDEX}/${SHARD_COUNT}:"
for t in "${selected_tests[@]}"; do
  echo "  - ${t}"
done
echo ""

# Build go test -run regex, matching full root test names.
regex="^($(printf '%s\n' "${selected_tests[@]}" | paste -sd '|' -))$"

# Use a per-shard JUnit file so CI can aggregate results.
junit_file="junit-report.xml"

gotestsum \
  --format short \
  --junitfile "${junit_file}" \
  --rerun-fails=3 \
  --rerun-fails-max-failures=1 \
  --packages="./test/e2e/..." \
  -- \
  -p 1 \
  -timeout 15m \
  -run "${regex}"
