#!/usr/bin/env bash
set -euo pipefail

# Shard e2e Go tests across CI workers.
# Usage (Semaphore example):
#   make test.e2e.autoparallel SHARD_INDEX=$SEMAPHORE_JOB_INDEX SHARD_COUNT=$SEMAPHORE_JOB_COUNT

source "$(dirname "${BASH_SOURCE[0]}")/lib/shard_args.sh"

echo "Running e2e tests shard ${SHARD_INDEX}/${SHARD_COUNT}"

if [[ ! -d "./test/e2e" ]]; then
  echo "No ./test/e2e directory found." >&2
  exit 1
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
  echo "No e2e tests found in ./test/e2e." >&2
  exit 1
fi

# Deduplicate and sort test names.
mapfile -t all_tests < <(printf '%s\n' "${all_tests[@]}" | sort -u)

# Timings only supply weights; source discovery remains authoritative. New
# tests get a conservative weight, and removed tests cannot enter a shard.
weighted_tests="$(printf '%s\n' "${all_tests[@]}" | awk '
  NR == FNR { if ($1 !~ /^#/ && NF == 2) durations[$1] = $2; next }
  { print (($1 in durations) ? durations[$1] : 60), $1 }
' "$(dirname "${BASH_SOURCE[0]}")/e2e_test_durations.tsv" - | sort -k1,1nr -k2,2)"

shard_loads=()
for ((shard=0; shard<SHARD_COUNT; shard++)); do
  shard_loads+=(0)
done
selected_tests=()
while read -r seconds test_name; do
  lightest=0
  for ((shard=1; shard<SHARD_COUNT; shard++)); do
    if (( shard_loads[shard] < shard_loads[lightest] )); then
      lightest="$shard"
    fi
  done
  shard_loads[$lightest]=$((shard_loads[lightest] + seconds))
  if [[ $((lightest + 1)) -eq "$SHARD_INDEX" ]]; then
    selected_tests+=("$test_name")
  fi
done <<< "$weighted_tests"

if [[ "${#selected_tests[@]}" -eq 0 ]]; then
  echo "No tests assigned to shard ${SHARD_INDEX}/${SHARD_COUNT}; exiting successfully."
  exit 0
fi

echo "Selected tests for shard ${SHARD_INDEX}/${SHARD_COUNT}:"
echo "Estimated test duration: ${shard_loads[$((SHARD_INDEX - 1))]}s"
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
  -timeout 30m \
  -run "${regex}"
