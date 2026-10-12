#!/bin/bash

set -euo pipefail
IFS=$'\n\t'

DB_NAME="${1:-superplane_dev}"

if [[ "$DB_NAME" != "superplane_dev" ]]; then
  echo "Local runner fleet setup only runs against superplane_dev." >&2
  exit 1
fi

export PGPASSWORD="${DB_PASSWORD:-the-cake-is-a-lie}"

psql \
  -v ON_ERROR_STOP=1 \
  -h "${DB_HOST:-db}" \
  -p "${DB_PORT:-5432}" \
  -U "${DB_USERNAME:-postgres}" \
  "$DB_NAME" <<'SQL'
INSERT INTO runner_fleets (
  id,
  scope_type,
  scope_id,
  slug,
  enabled,
  spec,
  runner_version,
  created_at,
  updated_at
)
VALUES (
  uuid_generate_v4(),
  'installation',
  NULL,
  'e1-large-amd64',
  true,
  jsonb_build_object(
    'operating_system', 'linux',
    'architecture', 'amd64',
    'cpu_millicores', 2000,
    'memory_mb', 8192,
    'disk_gb', 30,
    'capabilities', '[]'::jsonb
  ),
  'dev',
  now(),
  now()
)
ON CONFLICT (
  scope_type,
  (COALESCE(scope_id, '00000000-0000-0000-0000-000000000000'::uuid)),
  slug
)
WHERE deleted_at IS NULL
DO UPDATE SET
  enabled = EXCLUDED.enabled,
  spec = EXCLUDED.spec,
  runner_version = EXCLUDED.runner_version,
  updated_at = EXCLUDED.updated_at;
SQL
