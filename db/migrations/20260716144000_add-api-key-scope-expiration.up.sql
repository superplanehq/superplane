BEGIN;

ALTER TABLE users
  ADD COLUMN api_key_expires_at timestamp,
  ADD COLUMN api_key_canvas_ids jsonb NOT NULL DEFAULT '[]'::jsonb;

DROP INDEX IF EXISTS unique_service_account_in_organization;

UPDATE users
SET type = 'api_key'
WHERE type = 'service_account';

UPDATE casbin_rule
SET v2 = 'api_keys'
WHERE v2 = 'service_accounts';

CREATE UNIQUE INDEX unique_api_key_in_organization
  ON users (organization_id, name)
  WHERE type = 'api_key';

COMMIT;
