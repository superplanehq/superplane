BEGIN;

ALTER TABLE installation_github_apps
  ADD COLUMN client_id character varying(255) NOT NULL DEFAULT '',
  ADD COLUMN encrypted_client_secret bytea NOT NULL DEFAULT ''::bytea;

COMMIT;
