BEGIN;

CREATE TABLE account_choice_states (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  token_hash VARCHAR(64) NOT NULL,
  provider VARCHAR(64) NOT NULL,
  provider_id TEXT NOT NULL,
  redirect TEXT NOT NULL DEFAULT '',
  email TEXT NOT NULL DEFAULT '',
  name TEXT NOT NULL DEFAULT '',
  nickname TEXT NOT NULL DEFAULT '',
  avatar_url TEXT NOT NULL DEFAULT '',
  access_token BYTEA,
  refresh_token BYTEA,
  token_expires_at TIMESTAMPTZ,
  expires_at TIMESTAMPTZ NOT NULL,
  used_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT account_choice_states_token_hash_key UNIQUE (token_hash)
);

CREATE INDEX idx_account_choice_states_expires_at ON account_choice_states (expires_at);

COMMIT;
