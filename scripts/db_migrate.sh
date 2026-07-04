#!/bin/bash

set -euo pipefail
IFS=$'\n\t'

export DB_NAME=$1
export PGPASSWORD=the-cake-is-a-lie
DB_MIGRATION_LOCK_TIMEOUT="${DB_MIGRATION_LOCK_TIMEOUT:-600}"

rm -f db/structure.sql

migrate -lock-timeout "$DB_MIGRATION_LOCK_TIMEOUT" -source file://db/migrations -database "postgres://postgres:$PGPASSWORD@db:5432/$DB_NAME?sslmode=disable" up 2>&1 | awk '!/no change/'
migrate -lock-timeout "$DB_MIGRATION_LOCK_TIMEOUT" -source file://db/data_migrations -database "postgres://postgres:$PGPASSWORD@db:5432/$DB_NAME?sslmode=disable&x-migrations-table=data_migrations" up 2>&1 | awk '!/no change/'

pg_dump --schema-only --no-privileges --restrict-key abcdef123 --no-owner -h db -p 5432 -U postgres -d "$DB_NAME" > db/structure.sql
pg_dump --data-only --restrict-key abcdef123 --table schema_migrations -h db -p 5432 -U postgres -d "$DB_NAME" >> db/structure.sql
pg_dump --data-only --restrict-key abcdef123 --table data_migrations -h db -p 5432 -U postgres -d "$DB_NAME" >> db/structure.sql
