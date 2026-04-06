#!/usr/bin/env bash
set -euo pipefail

DB_NAME=$1

psql -h db -p 5432 -U postgres -c "ALTER DATABASE template1 REFRESH COLLATION VERSION" &>/dev/null
psql -h db -p 5432 -U postgres -c "ALTER DATABASE postgres REFRESH COLLATION VERSION" &>/dev/null
if createdb -h db -p 5432 -U postgres "$DB_NAME" 2>/dev/null; then
    echo "created $DB_NAME"
else
    echo "$DB_NAME already exists"
fi
psql -h db -p 5432 -U postgres "$DB_NAME" -c 'CREATE EXTENSION IF NOT EXISTS "uuid-ossp";' &>/dev/null
