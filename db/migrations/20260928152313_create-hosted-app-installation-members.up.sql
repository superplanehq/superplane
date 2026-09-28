BEGIN;

CREATE TABLE hosted_app_installation_members (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  provider TEXT NOT NULL,
  installation_id TEXT NOT NULL,
  member_login TEXT NOT NULL,
  allowed BOOLEAN NOT NULL DEFAULT FALSE,
  checked_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (provider, installation_id, member_login)
);

COMMIT;
