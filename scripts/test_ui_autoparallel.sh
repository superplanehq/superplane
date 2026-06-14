#!/usr/bin/env bash
set -euo pipefail

# Shard Vitest UI unit tests across CI workers.
# Usage (Semaphore example):
#   make check.test.ui.shard INDEX=$SEMAPHORE_JOB_INDEX TOTAL=$SEMAPHORE_JOB_COUNT
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

echo "Running UI unit tests shard ${INDEX}/${TOTAL}"

cd web_src
npm run test:run -- --shard="${INDEX}/${TOTAL}"
