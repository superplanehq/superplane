#!/bin/bash

set -euo pipefail
IFS=$'\n\t'

# Local-only. Writes a full dump of superplane_dev to
# .local/superplane_dev.dump so a later restore can skip owner setup
# and GitHub connection. Do not commit that file.

DUMP_PATH="${DUMP_PATH:-.local/superplane_dev.dump}"
DB_NAME="${1:-superplane_dev}"

if [[ "$DB_NAME" != "superplane_dev" ]]; then
  echo "db.snapshot only runs against superplane_dev." >&2
  exit 1
fi

export PGPASSWORD=the-cake-is-a-lie

mkdir -p "$(dirname "$DUMP_PATH")"
pg_dump -Fc --no-owner --no-acl \
  -h db -p 5432 -U postgres -d "$DB_NAME" \
  -f "$DUMP_PATH"

if [[ -n "${PUBLIC_API_PORT:-}" ]]; then
  echo "Wrote ${DUMP_PATH} from the database for http://localhost:${PUBLIC_API_PORT}."
else
  echo "Wrote ${DUMP_PATH}."
fi
