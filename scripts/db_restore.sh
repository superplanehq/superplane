#!/bin/bash

set -euo pipefail
IFS=$'\n\t'

# Local-only. Replaces superplane_dev from .local/superplane_dev.dump, then
# applies pending migrations. It does not touch superplane_test. It does not
# wipe Docker volumes.

DUMP_PATH="${DUMP_PATH:-.local/superplane_dev.dump}"
DB_NAME="${1:-superplane_dev}"

if [[ "$DB_NAME" != "superplane_dev" ]]; then
  echo "db.restore only runs against superplane_dev." >&2
  exit 1
fi

if [[ ! -f "$DUMP_PATH" ]]; then
  echo "Dump file ${DUMP_PATH} is missing." >&2
  echo "Set up the UI, then run make db.snapshot." >&2
  exit 1
fi

export PGPASSWORD=the-cake-is-a-lie

psql -h db -p 5432 -U postgres -d postgres -q -v ON_ERROR_STOP=1 \
  -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '${DB_NAME}' AND pid <> pg_backend_pid();"

dropdb -h db -p 5432 -U postgres --if-exists --force "$DB_NAME"
createdb -h db -p 5432 -U postgres "$DB_NAME"

pg_restore --no-owner --no-acl \
  -h db -p 5432 -U postgres -d "$DB_NAME" \
  "$DUMP_PATH"

./scripts/db_migrate.sh "$DB_NAME"

if [[ -n "${PUBLIC_API_PORT:-}" ]]; then
  echo "Restored ${DUMP_PATH} into ${DB_NAME} for http://localhost:${PUBLIC_API_PORT}."
else
  echo "Restored ${DUMP_PATH} into ${DB_NAME}."
fi
echo "Run make db.snapshot to refresh the local dump after migrations."
