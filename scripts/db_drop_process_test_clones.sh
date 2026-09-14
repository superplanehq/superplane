#!/bin/bash
set -euo pipefail
IFS=$'\n\t'

# Drop leftover per-process unit-test clones. Never drop superplane_dev or
# the shared template superplane_test.
#
# Clone names look like superplane_<pid>_test.

export PGPASSWORD="${DB_PASSWORD:-the-cake-is-a-lie}"

psql -h "${DB_HOST:-db}" -p "${DB_PORT:-5432}" -U "${DB_USERNAME:-postgres}" -d postgres -v ON_ERROR_STOP=1 <<'SQL'
DO $$
DECLARE
  dbname text;
BEGIN
  FOR dbname IN
    SELECT datname
    FROM pg_database
    WHERE datname ~ '^superplane_[0-9]+_test$'
  LOOP
    EXECUTE format('DROP DATABASE IF EXISTS %I WITH (FORCE)', dbname);
  END LOOP;
END
$$;
SQL
