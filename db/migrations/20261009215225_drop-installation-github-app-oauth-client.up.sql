BEGIN;

ALTER TABLE installation_github_apps
  DROP COLUMN client_id,
  DROP COLUMN encrypted_client_secret;

COMMIT;
