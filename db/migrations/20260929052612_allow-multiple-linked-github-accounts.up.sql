BEGIN;

ALTER TABLE account_linked_accounts
  ADD COLUMN active BOOLEAN NOT NULL DEFAULT FALSE;

-- The former unique index guarantees that each existing provider has one row.
UPDATE account_linked_accounts SET active = TRUE;

DROP INDEX idx_account_linked_accounts_account_provider;

CREATE UNIQUE INDEX idx_account_linked_accounts_account_provider_identity
  ON account_linked_accounts (account_id, provider, provider_id);

CREATE UNIQUE INDEX idx_account_linked_accounts_active_provider
  ON account_linked_accounts (account_id, provider)
  WHERE active;

COMMIT;
