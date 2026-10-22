BEGIN;

ALTER TABLE factories
  ADD COLUMN IF NOT EXISTS default_agent_model TEXT,
  ADD COLUMN IF NOT EXISTS default_agent_model_provider TEXT;

COMMIT;
