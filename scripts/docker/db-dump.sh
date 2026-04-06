#!/usr/bin/env bash
# Dumps schema and migration table data for a database to stdout.
# Usage: db-dump.sh DB_NAME [extra_data_tables...]
set -euo pipefail

DB_NAME=$1
shift

pg_dump --schema-only --no-privileges --restrict-key abcdef123 --no-owner \
    -h db -p 5432 -U postgres -d "$DB_NAME"

pg_dump --data-only --restrict-key abcdef123 --table schema_migrations \
    -h db -p 5432 -U postgres -d "$DB_NAME"

for table in "$@"; do
    pg_dump --data-only --restrict-key abcdef123 --table "$table" \
        -h db -p 5432 -U postgres -d "$DB_NAME"
done
