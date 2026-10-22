BEGIN;

ALTER TABLE mcp_oauth_codes
  ADD COLUMN factory_ids jsonb NOT NULL DEFAULT '[]'::jsonb;

UPDATE mcp_oauth_codes
SET factory_ids = jsonb_build_array(factory_id::text);

ALTER TABLE mcp_oauth_codes
  DROP COLUMN factory_id;

ALTER TABLE mcp_oauth_refresh_tokens
  ADD COLUMN factory_ids jsonb NOT NULL DEFAULT '[]'::jsonb;

UPDATE mcp_oauth_refresh_tokens
SET factory_ids = jsonb_build_array(factory_id::text);

ALTER TABLE mcp_oauth_refresh_tokens
  DROP COLUMN factory_id;

COMMIT;
