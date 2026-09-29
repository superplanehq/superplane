BEGIN;

ALTER TABLE hosted_app_installation_members
  ADD COLUMN errored BOOLEAN NOT NULL DEFAULT FALSE;

COMMIT;
