BEGIN;

ALTER TABLE accounts
  ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE;

CREATE INDEX index_accounts_on_deleted_at
  ON accounts (deleted_at);

COMMIT;
