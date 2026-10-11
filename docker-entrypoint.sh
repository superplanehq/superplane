#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

require_env() {
  local name="$1"

  if [ -z "${!name:-}" ]; then
    echo "${name} not set"
    exit 1
  fi
}

require_env DB_PASSWORD
require_env DB_HOST
require_env DB_PORT
require_env DB_USERNAME
require_env DB_NAME
require_env APPLICATION_NAME

DB_MIGRATION_LOCK_TIMEOUT="${DB_MIGRATION_LOCK_TIMEOUT:-600}"

[ "${POSTGRES_DB_SSL}" = "true" ] && export PGSSLMODE=require || export PGSSLMODE=disable

# Helm runs migrate in one hook Job (MIGRATE_ONLY=yes). Deployments start the
# binary directly and do not run this script. A second migrator that is
# killed while it holds the golang-migrate lock leaves schema_migrations dirty.
if [ "${RUN_MIGRATIONS:-yes}" = "yes" ]; then
  echo "Creating DB..."
  PGPASSWORD="${DB_PASSWORD}" createdb -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USERNAME}" "${DB_NAME}" || true

  echo "Migrating DB with ${DB_MIGRATION_LOCK_TIMEOUT}s lock timeout..."
  DB_URL="postgres://${DB_USERNAME}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${PGSSLMODE}"
  migrate -lock-timeout "${DB_MIGRATION_LOCK_TIMEOUT}" -source file:///app/db/migrations -database "${DB_URL}" up
  migrate -lock-timeout "${DB_MIGRATION_LOCK_TIMEOUT}" -source file:///app/db/data_migrations -database "${DB_URL}&x-migrations-table=data_migrations" up
else
  echo "Skipping DB migrate (RUN_MIGRATIONS=${RUN_MIGRATIONS})"
fi

if [ "${MIGRATE_ONLY:-no}" = "yes" ]; then
  echo "Migrate finished. Not starting the server."
  exit 0
fi

echo "Starting server..."
/app/build/superplane
