BEGIN;

-- The client credentials let a member authorize the public GitHub App, so
-- SuperPlane can open pull requests as that member. Sign-in does not use them.
ALTER TABLE installation_github_apps
  ADD COLUMN client_id character varying(255) NOT NULL DEFAULT '',
  ADD COLUMN encrypted_client_secret bytea NOT NULL DEFAULT ''::bytea;

COMMIT;
