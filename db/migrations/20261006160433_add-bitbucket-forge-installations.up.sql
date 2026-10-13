BEGIN;

CREATE TABLE bitbucket_forge_installations (
  installation_id      text NOT NULL,
  workspace_uuid       text NOT NULL DEFAULT '',
  workspace_slug       text NOT NULL DEFAULT '',
  installer_account_id text NOT NULL DEFAULT '',
  api_base_url         text NOT NULL DEFAULT '',
  system_token         bytea,
  token_expires_at     timestamp without time zone,
  last_delivery_at     timestamp without time zone,
  installed_at         timestamp without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  uninstalled_at       timestamp without time zone,
  created_at           timestamp without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at           timestamp without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT bitbucket_forge_installations_pkey PRIMARY KEY (installation_id)
);

CREATE INDEX bitbucket_forge_installations_workspace_uuid_idx
  ON bitbucket_forge_installations (workspace_uuid);

COMMIT;
