#!/usr/bin/env bash

shard_weight_names=()
shard_weight_values=()

# Load measured weights from a two-column file. Missing names use a fallback.
load_shard_weights() {
  local name weight
  shard_weight_names=()
  shard_weight_values=()
  while read -r name weight; do
    [[ "$name" == \#* || -z "$name" ]] && continue
    shard_weight_names+=("$name")
    shard_weight_values+=("$weight")
  done < "$1"
}

shard_weight_for() {
  local name="$1" fallback="$2" index
  for ((index = 0; index < ${#shard_weight_names[@]}; index++)); do
    if [[ "${shard_weight_names[$index]}" == "$name" ]]; then
      printf '%s\n' "${shard_weight_values[$index]}"
      return
    fi
  done
  printf '%s\n' "$fallback"
}

# Assign tab-separated weight/name pairs to 1-based shards. Equal weights and
# loads use a stable name/shard order across independent CI workers.
assign_weighted_shards() {
  local shard_count="$1"
  local -a loads=()
  local shard weight name lightest

  for ((shard = 0; shard < shard_count; shard++)); do
    loads+=(0)
  done

  while IFS=$'\t' read -r weight name; do
    [[ -n "$name" ]] || continue

    lightest=0
    for ((shard = 1; shard < shard_count; shard++)); do
      if ((loads[shard] < loads[lightest])); then
        lightest="$shard"
      fi
    done

    loads[lightest]=$((loads[lightest] + weight))
    printf '%s\t%s\n' "$((lightest + 1))" "$name"
  done < <(LC_ALL=C sort -t $'\t' -k1,1nr -k2,2)
}
