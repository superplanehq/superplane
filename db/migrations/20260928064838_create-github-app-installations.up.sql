BEGIN;

CREATE TABLE hosted_app_installations (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  provider TEXT NOT NULL,
  installation_id TEXT NOT NULL,
  account_login TEXT NOT NULL DEFAULT '',
  account_type TEXT NOT NULL DEFAULT '',
  account_id BIGINT NOT NULL DEFAULT 0,
  sender_login TEXT NOT NULL DEFAULT '',
  last_event_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ,
  UNIQUE (provider, installation_id)
);

CREATE INDEX idx_hosted_app_installations_account_login
  ON hosted_app_installations (provider, LOWER(account_login))
  WHERE deleted_at IS NULL;

COMMIT;
