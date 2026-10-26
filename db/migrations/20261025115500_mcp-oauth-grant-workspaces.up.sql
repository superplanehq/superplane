BEGIN;

ALTER TABLE mcp_oauth_codes
  ADD COLUMN factory_ids jsonb;

ALTER TABLE mcp_oauth_refresh_tokens
  ADD COLUMN factory_ids jsonb;

UPDATE mcp_oauth_codes
SET factory_ids = jsonb_build_array(factory_id::text)
WHERE factory_ids IS NULL;

UPDATE mcp_oauth_refresh_tokens
SET factory_ids = jsonb_build_array(factory_id::text)
WHERE factory_ids IS NULL;

COMMIT;
