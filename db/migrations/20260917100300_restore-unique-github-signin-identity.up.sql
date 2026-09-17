BEGIN;

DROP TABLE IF EXISTS account_choice_states;

DROP INDEX IF EXISTS idx_account_providers_provider_provider_id;
DROP INDEX IF EXISTS account_providers_non_github_provider_id_key;

-- Remove leftover provider rows for deleted accounts so they cannot occupy
-- a restored unique identity.
DELETE FROM account_providers
WHERE account_id IN (
  SELECT id FROM accounts WHERE deleted_at IS NOT NULL
);

-- Keep the oldest live account for each (provider, provider_id).
-- The preceding release allowed one GitHub identity on several accounts.
DELETE FROM account_providers
WHERE id IN (
  SELECT id FROM (
    SELECT ap.id,
           ROW_NUMBER() OVER (
             PARTITION BY ap.provider, ap.provider_id
             ORDER BY a.created_at ASC, a.id ASC, ap.created_at ASC, ap.id ASC
           ) AS rn
    FROM account_providers ap
    INNER JOIN accounts a ON a.id = ap.account_id
  ) ranked
  WHERE rn > 1
);

ALTER TABLE account_providers
  ADD CONSTRAINT account_providers_provider_provider_id_key
  UNIQUE (provider, provider_id);

COMMIT;
