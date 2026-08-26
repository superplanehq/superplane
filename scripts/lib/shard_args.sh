#!/usr/bin/env bash
# Shared shard argument handling for the test sharding scripts.
#
# Source this file, then use $SHARD_INDEX and $SHARD_COUNT.
#
#   SHARD_INDEX - 1-based index of this shard (defaults to 1)
#   SHARD_COUNT - total number of shards (defaults to 1)

SHARD_INDEX="${SHARD_INDEX:-1}"
SHARD_COUNT="${SHARD_COUNT:-1}"

if ! [[ "$SHARD_INDEX" =~ ^[0-9]+$ ]] || ! [[ "$SHARD_COUNT" =~ ^[0-9]+$ ]]; then
  echo "SHARD_INDEX and SHARD_COUNT must be positive integers (got SHARD_INDEX=${SHARD_INDEX}, SHARD_COUNT=${SHARD_COUNT})" >&2
  exit 1
fi

if [[ "$SHARD_COUNT" -lt 1 ]]; then
  echo "SHARD_COUNT must be >= 1 (got ${SHARD_COUNT})" >&2
  exit 1
fi

if [[ "$SHARD_INDEX" -lt 1 || "$SHARD_INDEX" -gt "$SHARD_COUNT" ]]; then
  echo "SHARD_INDEX must be between 1 and SHARD_COUNT (${SHARD_COUNT}), got ${SHARD_INDEX}" >&2
  exit 1
fi
