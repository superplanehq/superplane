#!/bin/bash

# Verify that database migrations added on the current branch are the latest.
#
# golang-migrate records only the highest applied version. Migrations run in
# timestamp order, so any migration with a timestamp lower than the recorded
# version is silently skipped.
#
# This script catches two failure modes:
# 1. Branch migrations older than the newest migration already on the base branch.
#    Production already applied main; those branch migrations are skipped on prod.
# 2. Base-branch migrations merged in after a branch migration was introduced.
#    Production already has those main migrations; branch migrations must be
#    re-timestamped so they remain the newest migrations in the repository.
#
# Branch migrations must therefore be the newest migrations in the repository
# at the time they are applied, and must stay newest after merges from main.
#

set -euo pipefail

BASE_BRANCH="${BASE_BRANCH:-main}"
migration_dirs=("db/migrations" "db/data_migrations")

function red() {
  echo -e "\033[0;31m$1\033[0m"
}

function yellow() {
  echo -e "\033[0;33m$1\033[0m"
}

function green() {
  echo -e "\033[0;32m$1\033[0m"
}

# Extract the normalized timestamp (first 14 digits: YYYYMMDDHHMMSS) from a
# migration file path. Prints nothing if the filename has no valid timestamp.
function timestamp_of() {
  local filename
  filename=$(basename "$1")
  local timestamp="${filename:0:14}"
  if [[ $timestamp =~ ^[0-9]{14}$ ]]; then
    echo "$timestamp"
  fi
}

function max_timestamp_from_files() {
  local max_timestamp=""
  local file timestamp

  for file in "$@"; do
    timestamp=$(timestamp_of "$file")
    [ -n "$timestamp" ] || continue
    if [ -z "$max_timestamp" ] || (( timestamp > max_timestamp )); then
      max_timestamp="$timestamp"
    fi
  done

  echo "$max_timestamp"
}

# Print migration files as "<basename> (timestamp: <ts>)", sorted by timestamp.
function format_migrations_with_timestamp() {
  local lines=()
  local file timestamp

  for file in "$@"; do
    timestamp=$(timestamp_of "$file")
    [ -n "$timestamp" ] || continue
    lines+=("${timestamp} $(basename "$file")")
  done

  [ ${#lines[@]} -eq 0 ] && return 0

  local sorted_line
  while IFS= read -r sorted_line; do
    [ -n "$sorted_line" ] || continue
    echo "${sorted_line#* } (timestamp: ${sorted_line%% *})"
  done < <(printf '%s\n' "${lines[@]}" | sort)
}

function earliest_commit() {
  local earliest=""
  local commit

  for commit in "$@"; do
    [ -n "$commit" ] || continue
    if [ -z "$earliest" ]; then
      earliest="$commit"
      continue
    fi
    if git merge-base --is-ancestor "$commit" "$earliest" 2>/dev/null; then
      earliest="$commit"
    fi
  done

  echo "$earliest"
}

# Collect branch migrations that golang-migrate would skip on production.
# Production follows main first, so it already records base_max_timestamp before
# this branch's migrations run. Any branch migration with a timestamp less than
# or equal to that version is silently skipped.
function collect_branch_migrations_skipped_on_prod() {
  local prod_version="$1"
  local skipped_lines=()

  for file in "${branch_up_migrations[@]}"; do
    local timestamp
    timestamp=$(timestamp_of "$file")
    [ -n "$timestamp" ] || continue
    (( timestamp > prod_version )) && continue
    skipped_lines+=("${timestamp} $(basename "$file")")
  done

  if [ ${#skipped_lines[@]} -eq 0 ]; then
    return 0
  fi

  local sorted_line
  while IFS= read -r sorted_line; do
    [ -n "$sorted_line" ] || continue
    echo "${sorted_line#* } (timestamp: ${sorted_line%% *})"
  done < <(printf '%s\n' "${skipped_lines[@]}" | sort)
}

function collect_interleaved_base_migrations() {
  local intro_commit="$1"
  local interleaved_lines=()

  while IFS= read -r file; do
    [[ "$file" == *.up.sql ]] || continue
    local timestamp
    timestamp=$(timestamp_of "$file")
    [ -n "$timestamp" ] || continue
    (( timestamp >= branch_max_timestamp )) && continue
    if git cat-file -e "${intro_commit}:${file}" 2>/dev/null; then
      continue
    fi
    interleaved_lines+=("${timestamp} $(basename "$file")")
  done < <(git ls-tree -r --name-only "$base_ref" -- "${migration_dirs[@]}" || true)

  if [ ${#interleaved_lines[@]} -eq 0 ]; then
    return 0
  fi

  local sorted_line
  while IFS= read -r sorted_line; do
    [ -n "$sorted_line" ] || continue
    echo "${sorted_line#* } (timestamp: ${sorted_line%% *})"
  done < <(printf '%s\n' "${interleaved_lines[@]}" | sort)
}

function report_branch_migrations_skipped_on_prod_error() {
  local prod_version="$1"
  local intro_commit="$2"
  local skipped_on_prod=()
  local interleaved_base=()
  local migration

  while IFS= read -r migration; do
    [ -n "$migration" ] && interleaved_base+=("$migration")
  done < <(collect_interleaved_base_migrations "$intro_commit")

  if [ ${#interleaved_base[@]} -eq 0 ]; then
    return 1
  fi

  while IFS= read -r migration; do
    [ -n "$migration" ] && skipped_on_prod+=("$migration")
  done < <(collect_branch_migrations_skipped_on_prod "$prod_version")

  red ""
  red "ERROR: Branch migrations were introduced before migrations merged from '${BASE_BRANCH}'!"
  red ""
  red "Production already applied '${BASE_BRANCH}' through version ${prod_version}."
  red "golang-migrate skips any migration with an older or equal timestamp."
  red ""
  red "These '${BASE_BRANCH}' migration(s) are already applied on production:"
  red ""
  for migration in "${interleaved_base[@]}"; do
    red "  - $migration"
  done
  red ""

  if [ ${#skipped_on_prod[@]} -gt 0 ]; then
    red "The following migration(s) added on this branch will be skipped on production:"
    red ""
    for migration in "${skipped_on_prod[@]}"; do
      red "  - $migration"
    done
    red ""
  else
    red "Recreate the branch migration(s) with a fresh timestamp after merging '${BASE_BRANCH}',"
    red "so they remain the newest migrations in the repository."
    red ""
  fi

  red "Migration(s) added on this branch (introduced in commit ${intro_commit:0:12}):"
  red ""
  while IFS= read -r migration; do
    [ -n "$migration" ] && red "  - $migration"
  done < <(format_migrations_with_timestamp "${branch_up_migrations[@]}")
  red ""

  red "Recreate the affected migration(s) with a fresh timestamp using:"
  red "    make db.migration.create NAME=<descriptive-name>"
  red ""

  return 0
}

# Resolve a usable ref for the base branch, fetching it if necessary.
function resolve_base_ref() {
  if git rev-parse --verify --quiet "origin/${BASE_BRANCH}" >/dev/null; then
    echo "origin/${BASE_BRANCH}"
    return 0
  fi

  if git rev-parse --verify --quiet "${BASE_BRANCH}" >/dev/null; then
    echo "${BASE_BRANCH}"
    return 0
  fi

  if git fetch --quiet origin "${BASE_BRANCH}" 2>/dev/null; then
    echo "FETCH_HEAD"
    return 0
  fi

  return 1
}

yellow "Checking that branch migrations are the latest (base branch: ${BASE_BRANCH})..."
yellow ""

base_ref=$(resolve_base_ref) || {
  yellow "Could not resolve base branch '${BASE_BRANCH}'; skipping check."
  green "✓ Nothing to verify"
  exit 0
}

merge_base=$(git merge-base "$base_ref" HEAD 2>/dev/null) || {
  yellow "Could not determine merge base with '${base_ref}'; skipping check."
  green "✓ Nothing to verify"
  exit 0
}

# Migration files added by commits on the current branch (since it diverged
# from the base branch). Retimestamped migrations remain in git history but
# are excluded once removed from HEAD.
branch_migrations=()
while IFS= read -r file; do
  [ -n "$file" ] || continue
  git cat-file -e "HEAD:${file}" 2>/dev/null || continue
  branch_migrations+=("$file")
done < <(git diff --name-only --diff-filter=A "$merge_base" HEAD -- \
  "db/migrations" "db/data_migrations" | grep '\.sql$' || true)

if [ ${#branch_migrations[@]} -eq 0 ]; then
  green "✓ No migrations added on this branch"
  exit 0
fi

# Migration files that exist on the base branch (these are the ones a
# branch migration must be newer than). Files added on this branch are not
# present on the base branch, so they are naturally excluded.
base_max_timestamp=""
base_max_file=""
while IFS= read -r file; do
  [[ "$file" == *.sql ]] || continue
  timestamp=$(timestamp_of "$file")
  [ -n "$timestamp" ] || continue
  if [ -z "$base_max_timestamp" ] || (( timestamp > base_max_timestamp )); then
    base_max_timestamp="$timestamp"
    base_max_file=$(basename "$file")
  fi
done < <(git ls-tree -r --name-only "$base_ref" -- "${migration_dirs[@]}" || true)

if [ -z "$base_max_timestamp" ]; then
  green "✓ No migrations on base branch to compare against"
  exit 0
fi

branch_up_migrations=()
for file in "${branch_migrations[@]}"; do
  [[ "$file" == *.up.sql ]] || continue
  branch_up_migrations+=("$file")
done

branch_max_timestamp=$(max_timestamp_from_files "${branch_up_migrations[@]}")
if [ -z "$branch_max_timestamp" ]; then
  green "✓ No timestamped branch migrations to verify"
  exit 0
fi

head_max_timestamp=""
head_max_file=""
while IFS= read -r file; do
  [[ "$file" == *.up.sql ]] || continue
  timestamp=$(timestamp_of "$file")
  [ -n "$timestamp" ] || continue
  if [ -z "$head_max_timestamp" ] || (( timestamp > head_max_timestamp )); then
    head_max_timestamp="$timestamp"
    head_max_file=$(basename "$file")
  fi
done < <(git ls-tree -r --name-only HEAD -- "${migration_dirs[@]}" || true)

# Verify every branch migration is newer than the newest base migration.
outdated_migrations=()
for file in "${branch_up_migrations[@]}"; do
  timestamp=$(timestamp_of "$file")
  [ -n "$timestamp" ] || continue
  if (( timestamp <= base_max_timestamp )); then
    outdated_migrations+=("$(basename "$file") (timestamp: $timestamp)")
  fi
done

if [ ${#outdated_migrations[@]} -gt 0 ]; then
  red ""
  red "ERROR: Found migration(s) on this branch that are not the latest!"
  red ""
  red "The base branch ('${BASE_BRANCH}') already has a newer migration:"
  red "  - ${base_max_file} (timestamp: ${base_max_timestamp})"
  red ""
  red "Production already applied '${BASE_BRANCH}' through version ${base_max_timestamp}."
  red "golang-migrate skips any migration with an older or equal timestamp."
  red ""
  red "The following migration(s) added on this branch will be skipped on production:"
  red ""

  for migration in "${outdated_migrations[@]}"; do
    red "  - $migration"
  done

  red ""
  red "Recreate the affected migration(s) with a fresh timestamp using:"
  red "    make db.migration.create NAME=<descriptive-name>"
  red ""

  exit 1
fi

if [ -n "$head_max_timestamp" ] && (( branch_max_timestamp < head_max_timestamp )); then
  red ""
  red "ERROR: Branch migrations are not the newest migrations in this repository!"
  red ""
  red "Newest migration on HEAD:"
  red "  - ${head_max_file} (timestamp: ${head_max_timestamp})"
  red ""
  red "Newest migration added on this branch:"
  red "  - timestamp: ${branch_max_timestamp}"
  red ""
  red "Recreate the branch migration(s) with a fresh timestamp using:"
  red "    make db.migration.create NAME=<descriptive-name>"
  red ""

  exit 1
fi

branch_intro_commits=()
for file in "${branch_up_migrations[@]}"; do
  intro_commit=$(git log --reverse --format=%H -- "$file" | head -1 || true)
  [ -n "$intro_commit" ] && branch_intro_commits+=("$intro_commit")
done

if [ ${#branch_intro_commits[@]} -eq 0 ]; then
  yellow "Could not determine when branch migrations were introduced; skipping merge-order check."
  green "✓ All branch migrations are newer than migrations on '${BASE_BRANCH}'"
  exit 0
fi

branch_intro_commit=$(earliest_commit "${branch_intro_commits[@]}")

if report_branch_migrations_skipped_on_prod_error \
  "$base_max_timestamp" \
  "$branch_intro_commit"; then
  exit 1
fi

green "✓ All branch migrations are newer than migrations on '${BASE_BRANCH}'"
exit 0
