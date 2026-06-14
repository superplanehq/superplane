#!/usr/bin/env bash
set -euo pipefail

# Shard Vitest UI unit tests across CI workers.
# Usage (Semaphore example):
#   make check.test.ui.shard INDEX=$SEMAPHORE_JOB_INDEX TOTAL=$SEMAPHORE_JOB_COUNT
#
# Environment:
#   INDEX - 0-based index of this shard (Semaphore default; converted to 1-based for Vitest)
#   TOTAL - total number of shards (defaults to 1)

TOTAL="${TOTAL:-${SEMAPHORE_JOB_COUNT:-1}}"
INDEX="${INDEX:-${SEMAPHORE_JOB_INDEX:-0}}"

if ! [[ "$INDEX" =~ ^[0-9]+$ ]] || ! [[ "$TOTAL" =~ ^[0-9]+$ ]]; then
  echo "INDEX and TOTAL must be non-negative integers (got INDEX=${INDEX}, TOTAL=${TOTAL})" >&2
  exit 1
fi

if [[ "$TOTAL" -lt 1 ]]; then
  echo "TOTAL must be >= 1 (got ${TOTAL})" >&2
  exit 1
fi

if [[ "$INDEX" -lt 0 || "$INDEX" -ge "$TOTAL" ]]; then
  echo "INDEX must be between 0 and TOTAL - 1 (${TOTAL} - 1), got ${INDEX}" >&2
  exit 1
fi

# Vitest --shard uses a 1-based index.
shard_index=$((INDEX + 1))

echo "Running UI unit tests shard ${shard_index}/${TOTAL}"

cd web_src
npm run test:run -- --shard="${shard_index}/${TOTAL}"
