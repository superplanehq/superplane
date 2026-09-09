BEGIN;

ALTER TABLE installation_llm_settings
  ADD COLUMN welcome_grant_ttl_days INTEGER NOT NULL DEFAULT 14;

ALTER TABLE installation_llm_settings
  ADD CONSTRAINT installation_llm_settings_welcome_ttl_positive
  CHECK (welcome_grant_ttl_days >= 1);

COMMIT;
