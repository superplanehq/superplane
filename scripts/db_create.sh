#!/bin/bash

set -euo pipefail
IFS=$'\n\t'

export DB_NAME=$1
export PGPASSWORD=the-cake-is-a-lie

psql -h db -p 5432 -U postgres -q -v ON_ERROR_STOP=1 -c "SET client_min_messages TO WARNING; ALTER DATABASE template1 REFRESH COLLATION VERSION;"
psql -h db -p 5432 -U postgres -q -v ON_ERROR_STOP=1 -c "SET client_min_messages TO WARNING; ALTER DATABASE postgres REFRESH COLLATION VERSION;"
createdb -h db -p 5432 -U postgres $DB_NAME > /dev/null 2>&1 || true
psql -h db -p 5432 -U postgres -q -v ON_ERROR_STOP=1 "$DB_NAME" -c "SET client_min_messages TO WARNING; CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";"