BEGIN;

ALTER TABLE users
  ADD COLUMN is_owner boolean NOT NULL DEFAULT false;

COMMIT;
