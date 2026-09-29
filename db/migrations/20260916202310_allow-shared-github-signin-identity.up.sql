BEGIN;

ALTER TABLE account_providers
  DROP CONSTRAINT account_providers_provider_provider_id_key;

CREATE UNIQUE INDEX account_providers_non_github_provider_id_key
  ON account_providers (provider, provider_id)
  WHERE provider <> 'github';

CREATE INDEX idx_account_providers_provider_provider_id
  ON account_providers (provider, provider_id);

COMMIT;
