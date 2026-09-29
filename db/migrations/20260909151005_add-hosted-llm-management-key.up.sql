BEGIN;

ALTER TABLE hosted_llm_providers
  ADD COLUMN IF NOT EXISTS management_key BYTEA;

COMMIT;
