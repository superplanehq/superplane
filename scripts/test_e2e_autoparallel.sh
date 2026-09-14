#!/usr/bin/env bash
set -euo pipefail

# Shard one E2E suite across CI workers.
# Usage:
#   E2E_DIR=./test/e2e/org E2E_PACKAGE=./test/e2e/org E2E_GO_PARALLEL=4 \
#     make test.e2e.org.autoparallel SHARD_INDEX=$SEMAPHORE_JOB_INDEX SHARD_COUNT=$SEMAPHORE_JOB_COUNT
#   E2E_DIR=./test/e2e/instance E2E_PACKAGE=./test/e2e/instance E2E_GO_PARALLEL=1 \
#     make test.e2e.instance.autoparallel SHARD_INDEX=$SEMAPHORE_JOB_INDEX SHARD_COUNT=$SEMAPHORE_JOB_COUNT

source "$(dirname "${BASH_SOURCE[0]}")/lib/shard_args.sh"

E2E_DIR="${E2E_DIR:-./test/e2e/org}"
E2E_PACKAGE="${E2E_PACKAGE:-${E2E_DIR}}"
E2E_GO_PARALLEL="${E2E_GO_PARALLEL:-4}"
E2E_TIMEOUT="${E2E_TIMEOUT:-15m}"

echo "Running e2e tests in ${E2E_DIR} shard ${SHARD_INDEX}/${SHARD_COUNT} (go -parallel ${E2E_GO_PARALLEL})"

if [[ ! -d "${E2E_DIR}" ]]; then
  echo "No ${E2E_DIR} directory found, nothing to run."
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
done < <(find "${E2E_DIR}" -maxdepth 1 -type f -name '*_test.go' | sort)

if [[ "${#all_tests[@]}" -eq 0 ]]; then
  echo "No e2e tests found in ${E2E_DIR}, nothing to run."
  exit 0
fi

# Deduplicate and sort test names.
mapfile -t all_tests < <(printf '%s\n' "${all_tests[@]}" | sort -u)

selected_tests=()
idx=0
for test_name in "${all_tests[@]}"; do
  shard=$(( (idx % SHARD_COUNT) + 1 ))
  if [[ "$shard" -eq "$SHARD_INDEX" ]]; then
    selected_tests+=("$test_name")
  fi
  idx=$((idx + 1))
done

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
  --packages="${E2E_PACKAGE}" \
  -- \
  -p 1 \
  -parallel "${E2E_GO_PARALLEL}" \
  -timeout "${E2E_TIMEOUT}" \
  -run "${regex}"
