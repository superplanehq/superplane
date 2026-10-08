BEGIN;

CREATE TABLE installation_github_apps (
  id                        integer NOT NULL,
  github_app_id             bigint NOT NULL,
  slug                      character varying(255) NOT NULL,
  encrypted_private_key     bytea NOT NULL,
  encrypted_webhook_secret  bytea NOT NULL,
  created_at                timestamp without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at                timestamp without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT installation_github_apps_pkey PRIMARY KEY (id),
  CONSTRAINT installation_github_apps_singleton CHECK (id = 1)
);

COMMIT;
