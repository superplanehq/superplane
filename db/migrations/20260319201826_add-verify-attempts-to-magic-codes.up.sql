BEGIN;

ALTER TABLE account_magic_codes ADD COLUMN verify_attempts INTEGER NOT NULL DEFAULT 0;

COMMIT;
