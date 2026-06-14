#!/bin/bash

# Verify that database migrations added on the current branch are the latest.
#
# When a branch adds DB migrations, those migrations get a timestamp at creation
# time. If another branch creates migrations later but gets merged to the base
# branch first, the migrations on the current branch end up with timestamps that
# are earlier than migrations already applied in production.
#
# Because migrations are applied in timestamp order and the migration runner
# tracks the latest applied version, migrations on the current branch with
# timestamps older than the most recent already-applied migration can be
# silently skipped on prod. This script catches that situation in CI.
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
# from the base branch).
branch_migrations=()
while IFS= read -r file; do
  [ -n "$file" ] && branch_migrations+=("$file")
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

# Verify every branch migration is newer than the newest base migration.
outdated_migrations=()
for file in "${branch_migrations[@]}"; do
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
  red "The following migration(s) on this branch have an older (or equal) timestamp:"
  red ""

  for migration in "${outdated_migrations[@]}"; do
    red "  - $migration"
  done

  red ""
  red "Migrations are applied in timestamp order, so a migration with a timestamp"
  red "older than one already applied on the base branch can be silently skipped"
  red "in production. Please recreate the affected migration(s) with a fresh"
  red "timestamp using:"
  red "    make db.migration.create NAME=<descriptive-name>"
  red ""

  exit 1
fi

green "✓ All branch migrations are newer than migrations on '${BASE_BRANCH}'"
exit 0
