BEGIN;

DROP INDEX IF EXISTS unique_service_account_in_organization;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name = 'users'
      AND column_name = 'service_account_expires_at'
  ) AND NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name = 'users'
      AND column_name = 'api_key_expires_at'
  ) THEN
    ALTER TABLE users
      RENAME COLUMN service_account_expires_at TO api_key_expires_at;
  END IF;
END $$;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name = 'users'
      AND column_name = 'service_account_canvas_ids'
  ) AND NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name = 'users'
      AND column_name = 'api_key_canvas_ids'
  ) THEN
    ALTER TABLE users
      RENAME COLUMN service_account_canvas_ids TO api_key_canvas_ids;
  END IF;
END $$;

UPDATE users
SET type = 'api_key'
WHERE type = 'service_account';

UPDATE casbin_rule
SET v2 = 'api_keys'
WHERE v2 = 'service_accounts';

CREATE UNIQUE INDEX IF NOT EXISTS unique_api_key_in_organization
  ON users (organization_id, name)
  WHERE type = 'api_key';

COMMIT;
