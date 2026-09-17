BEGIN;

DROP TABLE IF EXISTS account_choice_states;

DROP INDEX IF EXISTS idx_account_providers_provider_provider_id;
DROP INDEX IF EXISTS account_providers_non_github_provider_id_key;

ALTER TABLE account_providers
  ADD CONSTRAINT account_providers_provider_provider_id_key
  UNIQUE (provider, provider_id);

COMMIT;
