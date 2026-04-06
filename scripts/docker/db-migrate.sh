#!/usr/bin/env bash
set -euo pipefail

DB_NAME=$1
MIGRATIONS_DIR=$2
DATA_MIGRATIONS_DIR=${3:-}

DB_URL="postgres://postgres:${PGPASSWORD}@db:5432/${DB_NAME}?sslmode=disable"

migrate -source "file://${MIGRATIONS_DIR}" -database "$DB_URL" up

if [ -n "$DATA_MIGRATIONS_DIR" ]; then
    migrate -source "file://${DATA_MIGRATIONS_DIR}" -database "${DB_URL}&x-migrations-table=data_migrations" up
fi
