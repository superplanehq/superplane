#!/usr/bin/env bash
set -euo pipefail

# Shard Vitest UI unit tests across CI workers.
# Usage (Semaphore example):
#   make check.test.ui.shard SHARD_INDEX=$SEMAPHORE_JOB_INDEX SHARD_COUNT=$SEMAPHORE_JOB_COUNT

source "$(dirname "${BASH_SOURCE[0]}")/lib/shard_args.sh"

echo "Running UI unit tests shard ${SHARD_INDEX}/${SHARD_COUNT}"

cd web_src
npm run test:run -- --shard="${SHARD_INDEX}/${SHARD_COUNT}"
