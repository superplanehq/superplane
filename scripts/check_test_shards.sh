#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/lib/weighted_shards.sh"

assert_equal() {
  if [[ "$1" != "$2" ]]; then
    printf 'Expected:\n%s\nActual:\n%s\n' "$2" "$1" >&2
    exit 1
  fi
}

assert_complete_assignment() {
  local input="$1" assignment="$2" shard_count="$3"
  local expected_names actual_names shard name

  expected_names="$(printf '%s\n' "$input" | cut -f2 | LC_ALL=C sort)"
  actual_names="$(printf '%s\n' "$assignment" | cut -f2 | LC_ALL=C sort)"
  assert_equal "$actual_names" "$expected_names"

  while IFS=$'\t' read -r shard name; do
    if [[ -z "$name" || "$shard" -lt 1 || "$shard" -gt "$shard_count" ]]; then
      printf 'Invalid shard assignment: %s\t%s\n' "$shard" "$name" >&2
      exit 1
    fi
  done <<< "$assignment"
}

weights=$'9\tslow\n3\talpha\n3\tbeta\n1\tgamma'
shuffled=$'1\tgamma\n3\tbeta\n9\tslow\n3\talpha'
assignment="$(printf '%s\n' "$weights" | assign_weighted_shards 2)"
assert_complete_assignment "$weights" "$assignment" 2
assert_equal "$(printf '%s\n' "$shuffled" | assign_weighted_shards 2)" "$assignment"
assert_equal "$assignment" $'1\tslow\n2\talpha\n2\tbeta\n2\tgamma'

few_items=$'2\talpha\n1\tbeta'
assignment="$(printf '%s\n' "$few_items" | assign_weighted_shards 5)"
assert_complete_assignment "$few_items" "$assignment" 5
assert_equal "$assignment" $'1\talpha\n2\tbeta'

load_shard_weights <(printf '# measured weights\nslow 9\n')
assert_equal "$(shard_weight_for slow 30)" 9
assert_equal "$(shard_weight_for new_test 30)" 30

echo "Weighted shard assignment checks passed."
