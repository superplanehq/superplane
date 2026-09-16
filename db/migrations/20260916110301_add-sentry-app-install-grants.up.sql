BEGIN;

CREATE TABLE sentry_app_install_grants (
  installation_uuid  TEXT PRIMARY KEY,
  code_digest        TEXT NOT NULL,
  organization_slug  TEXT NOT NULL DEFAULT '',
  access_token       BYTEA,
  refresh_token      BYTEA,
  token_expires_at   TEXT NOT NULL DEFAULT '',
  expires_at         TIMESTAMPTZ NOT NULL,
  created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sentry_app_install_grants_expires_at
  ON sentry_app_install_grants (expires_at);

COMMIT;
