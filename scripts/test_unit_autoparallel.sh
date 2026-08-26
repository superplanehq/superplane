#!/usr/bin/env bash
set -euo pipefail

# Shard Go unit tests across CI workers.
# Usage (Semaphore example):
#   make test.coverage.autoparallel INDEX=$SEMAPHORE_JOB_INDEX TOTAL=$SEMAPHORE_JOB_COUNT
#
# Environment:
#   INDEX - 1-based index of this shard (defaults to 1)
#   TOTAL - total number of shards (defaults to 1)

INDEX="${INDEX:-${SEMAPHORE_JOB_INDEX:-1}}"
TOTAL="${TOTAL:-${SEMAPHORE_JOB_COUNT:-1}}"

if ! [[ "$INDEX" =~ ^[0-9]+$ ]] || ! [[ "$TOTAL" =~ ^[0-9]+$ ]]; then
  echo "INDEX and TOTAL must be positive integers (got INDEX=${INDEX}, TOTAL=${TOTAL})" >&2
  exit 1
fi

if [[ "$TOTAL" -lt 1 ]]; then
  echo "TOTAL must be >= 1 (got ${TOTAL})" >&2
  exit 1
fi

if [[ "$INDEX" -lt 1 || "$INDEX" -gt "$TOTAL" ]]; then
  echo "INDEX must be between 1 and TOTAL (${TOTAL}), got ${INDEX}" >&2
  exit 1
fi

echo "Running unit tests shard ${INDEX}/${TOTAL}"

module_prefix="$(go list -m)"
mapfile -t all_packages < <(go list ./pkg/...)

if [[ "${#all_packages[@]}" -eq 0 ]]; then
  echo "No packages found under ./pkg, nothing to run."
  exit 0
fi

# Balance shards by *_test.go count so large packages do not cluster on one job.
package_weights=()
for import_path in "${all_packages[@]}"; do
  rel="./${import_path#"${module_prefix}/"}"
  weight="$(find "$rel" -maxdepth 1 -type f -name '*_test.go' 2>/dev/null | wc -l | tr -d ' ')"
  package_weights+=("${weight}:${import_path}")
done

mapfile -t package_weights < <(printf '%s\n' "${package_weights[@]}" | sort -t: -k1,1nr -k2,2)

shard_weights=()
for ((i = 0; i < TOTAL; i++)); do
  shard_weights+=("0")
done

selected_packages=()
for entry in "${package_weights[@]}"; do
  weight="${entry%%:*}"
  import_path="${entry#*:}"
  lightest_shard=0
  lightest_weight="${shard_weights[0]}"
  for ((i = 1; i < TOTAL; i++)); do
    if [[ "${shard_weights[$i]}" -lt "$lightest_weight" ]]; then
      lightest_shard="$i"
      lightest_weight="${shard_weights[$i]}"
    fi
  done

  shard_index=$((lightest_shard + 1))
  shard_weights[$lightest_shard]=$((lightest_weight + weight))
  if [[ "$shard_index" -eq "$INDEX" ]]; then
    selected_packages+=("${import_path}")
  fi
done

if [[ "${#selected_packages[@]}" -eq 0 ]]; then
  echo "No packages assigned to shard ${INDEX}/${TOTAL}; exiting successfully."
  exit 0
fi

selected_rel=()
echo "Selected packages for shard ${INDEX}/${TOTAL}:"
for import_path in "${selected_packages[@]}"; do
  rel="./${import_path#"${module_prefix}/"}"
  selected_rel+=("$rel")
  echo "  - ${rel}"
done
echo ""

packages_csv="$(IFS=','; echo "${selected_rel[*]}")"

gotestsum \
  --format short \
  --junitfile junit-report.xml \
  --packages="${packages_csv}" \
  -- \
  -p 1 \
  -coverprofile=coverage-go.out \
  -covermode=atomic \
  -coverpkg="${packages_csv}"

go tool cover -func=coverage-go.out | grep '^total:'

coverage_args=(--profile coverage-go.out)
if [[ "$TOTAL" -gt 1 ]]; then
  coverage_args+=(--ignore-missing-packages)
fi

go run ./scripts/check_go_coverage_budget.go "${coverage_args[@]}"
