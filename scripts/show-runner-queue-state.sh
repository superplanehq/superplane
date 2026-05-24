#!/usr/bin/env bash
# Show correlated task rows from task-broker (PostgreSQL) + fleet-manager (SQLite on remote host).
#
# Broker queries use TASK_BROKER_DATABASE_URL locally (psql). Fleet-manager still uses SSH +
# alpine sqlite against the remote Docker volume.
#
# Env:
#   TASK_BROKER_DATABASE_URL   — required for broker rows (postgres connection string)
#   TASK_BROKER_SSH_HOST       — default 98.91.210.215 (informational)
#   FLEET_MANAGER_SSH_HOST     — default 13.220.51.216
#   SSH_USER                   — default ubuntu
#   SSH_KEY                    — default ~/.ssh/igor-runners.pem
#   SHOW_RUNNER_TASK_LIMIT     — broker rows (default 25); digits only
#
# Usage:
#   TASK_BROKER_DATABASE_URL='postgres://…' ./scripts/show-runner-queue-state.sh

set -euo pipefail

TASK_BROKER_SSH_HOST="${TASK_BROKER_SSH_HOST:-98.91.210.215}"
FLEET_MANAGER_SSH_HOST="${FLEET_MANAGER_SSH_HOST:-13.220.51.216}"
SSH_USER="${SSH_USER:-ubuntu}"
SSH_KEY="${SSH_KEY:-$HOME/.ssh/igor-runners.pem}"
SSH_KEY="${SSH_KEY/#\~/$HOME}"

LIMIT="${SHOW_RUNNER_TASK_LIMIT:-25}"
if ! [[ "${LIMIT}" =~ ^[1-9][0-9]*$ ]]; then
  echo "SHOW_RUNNER_TASK_LIMIT must be a positive integer" >&2
  exit 2
fi

if [[ ! -f "$SSH_KEY" ]]; then
  echo "SSH key not found: $SSH_KEY" >&2
  exit 1
fi

if [[ -z "${TASK_BROKER_DATABASE_URL:-}" ]]; then
  echo "TASK_BROKER_DATABASE_URL is required (task-broker uses PostgreSQL)" >&2
  exit 1
fi

if ! command -v psql >/dev/null 2>&1; then
  echo "psql not found on PATH (required for broker queries)" >&2
  exit 1
fi

SSH_FLEET=(ssh -i "$SSH_KEY" -o StrictHostKeyChecking=accept-new "${SSH_USER}@${FLEET_MANAGER_SSH_HOST}")

sql_b64() {
  printf '%s' "$1" | base64 | tr -d '\n\r'
}

broker_sql() {
  psql "$TASK_BROKER_DATABASE_URL" -At -v ON_ERROR_STOP=1 -c "$1"
}

fleet_sql_remote() {
  local enc
  enc="$(sql_b64 "$1")"
  "${SSH_FLEET[@]}" "echo '${enc}' | base64 -d | sudo docker run -i --rm -v fleet-manager-data:/data alpine:3.20 sh -lc 'apk add -q sqlite >/dev/null && sqlite3 -batch /data/fleet.db'"
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

echo "TASK_BROKER:    ${TASK_BROKER_SSH_HOST}"
echo "FLEET_MANAGER:  ${FLEET_MANAGER_SSH_HOST}"
echo ""

BROKER_QUERY=$(
  cat <<EOSQL
SELECT id || '|' || fleet_id || '|' || COALESCE(fleet_task_id, '') || '|' || FLOOR(EXTRACT(EPOCH FROM created_at))::bigint
FROM broker_tasks ORDER BY created_at DESC LIMIT ${LIMIT};
EOSQL
)

broker_lines=()
while IFS= read -r _line || [[ -n "${_line}" ]]; do broker_lines+=("$_line"); done < <(broker_sql "$BROKER_QUERY" | tr -d '\r')

if [[ "${#broker_lines[@]}" -eq 0 ]]; then
  echo "No broker_tasks rows returned (broker DB empty or SSH/SQL failure)." >&2
  exit 1
fi

print_header() {
  printf '%-38s  %-34s  %-38s  %-18s  %-14s  %-22s  %-18s  %s\n' \
    'broker_task_id' \
    'fleet_id' \
    'fleet_task_id' \
    'created_utc' \
    'fleet_status' \
    'runner_id' \
    'lease_until_utc' \
    'note'
}

divider() {
  printf '%230s\n' '' | tr ' ' -
}

print_header
divider

for row in "${broker_lines[@]}"; do
  [[ -z "$row" ]] && continue
  IFS='|' read -r broker_id fleet_id fleet_task_id created_unix <<<"${row}"

  ts="$(fmt_unix "$created_unix")"
  fleet_status="-"
  runner_id="-"
  lease_note="-"
  note=""

  if [[ -z "$fleet_task_id" ]]; then
    note="fleet_task unset (broker still correlating upstream id)"
    printf '%-38s  %-34s  %-38s  %-18s  %-14s  %-22s  %-18s  %s\n' \
      "$broker_id" "$fleet_id" "(empty)" "$ts" "-" "-" "-" "$note"
    continue
  fi

  safe_fid="${fleet_task_id//\'/}"
  FLEET_QUERY=$(
    cat <<EOSQL
SELECT status || '|' || COALESCE(runner_id,'') || '|' || CASE WHEN lease_until IS NULL THEN '' ELSE CAST(lease_until AS TEXT) END
FROM tasks WHERE id='${safe_fid}';
EOSQL
  )

  fleet_line="$(fleet_sql_remote "$FLEET_QUERY" | tr -d '\r')"
  fleet_line="$(echo "$fleet_line" | head -1)"

  if [[ -z "$fleet_line" ]]; then
    fleet_status="(missing)"
    note="no fleet.tasks row — wrong fleet DB or orphaned broker row"
  else
    IFS='|' read -r fleet_status runner_id lease_unix <<<"${fleet_line}"
    lease_human="$(fmt_unix "${lease_unix:-}")"
    lease_note="$lease_human"
    case "${fleet_status}" in
      queued)
        note="no runner claimed yet (need POST claim + AUTH_TOKEN if fleet uses bearer)"
        ;;
      claimed|running)
        note="runner has lease — if stuck, check runner logs / expiry"
        ;;
      succeeded|failed|timeout|canceled)
        ;;
      *)
        ;;
    esac
  fi

  printf '%-38s  %-34s  %-38s  %-18s  %-14s  %-22s  %-18s  %s\n' \
    "$broker_id" "$fleet_id" "$fleet_task_id" "$ts" "$fleet_status" "$runner_id" "$lease_note" "$note"
done

echo ""
echo "Fleet tasks by status (${FLEET_MANAGER_SSH_HOST} fleet.db):"
SUMMARY_SQL='SELECT status, COUNT(*) FROM tasks GROUP BY status ORDER BY COUNT(*) DESC;'
fleet_sql_remote "$SUMMARY_SQL" | tr -d '\r' | while IFS= read -r line; do
  printf '  %s\n' "${line}"

done || true

echo ""
echo "Broker rows by fleet_task linkage (PostgreSQL):"
LINK_SQL="SELECT CASE WHEN fleet_task_id IS NULL OR fleet_task_id='' THEN 'no_fleet_task_id' ELSE 'has_fleet_task_id' END AS k, COUNT(*) FROM broker_tasks GROUP BY k;"
broker_sql "$LINK_SQL" | tr -d '\r' | while IFS= read -r line; do
  printf '  %s\n' "${line}"

done || true
