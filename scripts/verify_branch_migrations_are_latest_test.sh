#!/usr/bin/env bash
# Exercise migration ordering against real Git branches and merges.
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
fixture="$(mktemp -d)"
trap 'rm -rf "$fixture"' EXIT

assert_status() {
  local expected="$1" name="$2" status=0
  BASE_BRANCH=main bash "$repo_root/scripts/verify_branch_migrations_are_latest.sh" >"$fixture/output" 2>&1 || status=$?
  if [ "$status" -ne "$expected" ]; then
    cat "$fixture/output" >&2
    echo "FAIL $name: expected status $expected, got $status" >&2
    exit 1
  fi
  if [ "$expected" -ne 0 ]; then
    grep -q 'will be skipped on production' "$fixture/output"
  fi
  echo "PASS $name"
}

for migration_dir in db/migrations db/data_migrations; do
  mkdir "$fixture/$(basename "$migration_dir")"
  cd "$fixture/$(basename "$migration_dir")"
  git init -q --initial-branch=main
  git config user.name 'Migration test'
  git config user.email 'migration-test@example.com'
  git config commit.gpgsign false
  git config core.hooksPath /dev/null
  mkdir -p "$migration_dir"
  touch "$migration_dir/20260101000000_base.up.sql"
  git add .
  git commit -qm 'Add base migration'
  assert_status 0 "$migration_dir: no branch migrations"

  git checkout -qb feature
  touch "$migration_dir/20260103000000_feature.up.sql"
  git add .
  git commit -qm 'Add feature migration'

  git checkout -q main
  touch "$migration_dir/20260102000000_main.up.sql"
  git add .
  git commit -qm 'Add earlier main migration'
  git checkout -q feature
  git merge -q --no-edit main
  assert_status 0 "$migration_dir: newer migration survives merge from main"

  git checkout -q main
  touch "$migration_dir/20260103000000_equal.up.sql"
  git add .
  git commit -qm 'Add equal main migration'
  git checkout -q feature
  git merge -q --no-edit main
  assert_status 1 "$migration_dir: equal migration is rejected"

  git checkout -q main
  touch "$migration_dir/20260104000000_newer.up.sql"
  git add .
  git commit -qm 'Add newer main migration'
  git checkout -q feature
  git merge -q --no-edit main
  touch "$migration_dir/20260105000000_latest.up.sql"
  git add .
  git commit -qm 'Add another feature migration'
  assert_status 1 "$migration_dir: older migration is rejected even with a newer one"
done
