BEGIN;

UPDATE factory_agent_resources
SET config = jsonb_set(config, '{toolsDefaultApplied}', 'true'::jsonb, true)
WHERE kind = 'mcp_server';

COMMIT;
