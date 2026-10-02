#!/bin/sh

set -eu

if [ -n "${INSTALLATION_ADMIN_TOKEN:-}" ]; then
  exec /usr/local/bin/fleet-manager "$@"
fi

database_host="${DB_HOST:-db}"
database_port="${DB_PORT:-5432}"
database_name="${DB_NAME:-superplane_dev}"
database_user="${DB_USERNAME:-postgres}"
database_password="${DB_PASSWORD:-the-cake-is-a-lie}"
token_name="Local Fleet Manager"
token="${LOCAL_FLEET_MANAGER_ADMIN_TOKEN:-superplane-local-fleet-manager-token}"
token_hash="$(printf '%s' "$token" | sha256sum | cut -d ' ' -f 1)"

query_database() {
  PGPASSWORD="$database_password" psql \
    -X \
    -v ON_ERROR_STOP=1 \
    -h "$database_host" \
    -p "$database_port" \
    -U "$database_user" \
    -d "$database_name" \
    "$@"
}

echo "Waiting for the local owner account."
owner_user_id=""
while [ -z "$owner_user_id" ]; do
  owner_user_id="$(
    query_database -Atq -c "
      SELECT users.id
      FROM users
      INNER JOIN accounts ON accounts.id = users.account_id
      WHERE accounts.installation_admin = true
        AND accounts.blocked_at IS NULL
        AND accounts.deleted_at IS NULL
        AND users.type = 'human'
        AND users.deleted_at IS NULL
      ORDER BY accounts.created_at ASC NULLS LAST, users.created_at ASC
      LIMIT 1
    " 2>/dev/null || true
  )"
  if [ -z "$owner_user_id" ]; then
    sleep 2
  fi
done

query_database \
  -v owner_user_id="$owner_user_id" \
  -v token_hash="$token_hash" \
  -v token_name="$token_name" <<'SQL'
BEGIN;

WITH selected AS (
  SELECT id
  FROM user_api_tokens
  WHERE name = :'token_name'
  ORDER BY created_at ASC, id ASC
  LIMIT 1
)
DELETE FROM user_api_tokens
WHERE name = :'token_name'
  AND id <> (SELECT id FROM selected);

UPDATE user_api_tokens
SET
  user_id = :'owner_user_id'::uuid,
  token_hash = :'token_hash',
  last_used_at = NULL
WHERE id = (
  SELECT id
  FROM user_api_tokens
  WHERE name = :'token_name'
  ORDER BY created_at ASC, id ASC
  LIMIT 1
);

INSERT INTO user_api_tokens (user_id, name, token_hash)
SELECT :'owner_user_id'::uuid, :'token_name', :'token_hash'
WHERE NOT EXISTS (
  SELECT 1
  FROM user_api_tokens
  WHERE name = :'token_name'
);

COMMIT;
SQL

echo "Local Fleet Manager credentials are ready."
export INSTALLATION_ADMIN_TOKEN="$token"
exec /usr/local/bin/fleet-manager "$@"
