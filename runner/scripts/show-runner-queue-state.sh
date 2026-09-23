#!/usr/bin/env bash
# Show recent task rows from task-broker (PostgreSQL).
#
# Env:
#   TASK_BROKER_DATABASE_URL   — required (postgres connection string)
#   TASK_BROKER_SSH_HOST       — default 98.91.210.215 (informational)
#   SHOW_RUNNER_TASK_LIMIT     — rows (default 25); digits only
#
# Usage:
#   TASK_BROKER_DATABASE_URL='postgres://…' ./scripts/show-runner-queue-state.sh

set -euo pipefail

TASK_BROKER_SSH_HOST="${TASK_BROKER_SSH_HOST:-98.91.210.215}"
LIMIT="${SHOW_RUNNER_TASK_LIMIT:-25}"
if ! [[ "${LIMIT}" =~ ^[1-9][0-9]*$ ]]; then
  echo "SHOW_RUNNER_TASK_LIMIT must be a positive integer" >&2
  exit 2
fi

if [[ -z "${TASK_BROKER_DATABASE_URL:-}" ]]; then
  echo "TASK_BROKER_DATABASE_URL is required (task-broker uses PostgreSQL)" >&2
  exit 1
fi

if ! command -v psql >/dev/null 2>&1; then
  echo "psql not found on PATH (required for broker queries)" >&2
  exit 1
fi

broker_sql() {
  psql "$TASK_BROKER_DATABASE_URL" -At -v ON_ERROR_STOP=1 -c "$1"
}

fmt_unix() {
  local ts="$1"
  if [[ -z "${ts:-}" || "$ts" == "NULL" ]]; then
    echo "-"
    return
  fi
  if [[ "$OSTYPE" == darwin* ]]; then
    date -u -r "$ts" +'%Y-%m-%d %H:%M UTC' 2>/dev/null || echo "$ts"
  else
    date -u -d "@${ts}" +'%Y-%m-%d %H:%M UTC' 2>/dev/null || echo "$ts"
  fi
}

echo "TASK_BROKER: ${TASK_BROKER_SSH_HOST}"
echo ""

BROKER_QUERY=$(
  cat <<EOSQL
SELECT id || '|' || fleet_id || '|' || status || '|' || COALESCE(runner_id,'') || '|' ||
  CASE WHEN lease_until IS NULL THEN '' ELSE FLOOR(EXTRACT(EPOCH FROM lease_until))::bigint::text END || '|' ||
  FLOOR(EXTRACT(EPOCH FROM created_at))::bigint
FROM tasks ORDER BY created_at DESC LIMIT ${LIMIT};
EOSQL
)

print_header() {
  printf '%-38s  %-18s  %-14s  %-22s  %-18s  %-18s\n' \
    'task_id' 'fleet_id' 'status' 'runner_id' 'lease_until_utc' 'created_utc'
}

divider() {
  printf '%130s\n' '' | tr ' ' -
}

print_header
divider

while IFS= read -r row || [[ -n "${row}" ]]; do
  [[ -z "$row" ]] && continue
  IFS='|' read -r task_id fleet_id status runner_id lease_unix created_unix <<<"${row}"
  printf '%-38s  %-18s  %-14s  %-22s  %-18s  %-18s\n' \
    "$task_id" "$fleet_id" "$status" "${runner_id:--}" "$(fmt_unix "$lease_unix")" "$(fmt_unix "$created_unix")"
done < <(broker_sql "$BROKER_QUERY" | tr -d '\r')

echo ""
echo "Tasks by status:"
broker_sql 'SELECT status, COUNT(*) FROM tasks GROUP BY status ORDER BY COUNT(*) DESC;' | tr -d '\r' | while IFS= read -r line; do
  printf '  %s\n' "${line}"
done
