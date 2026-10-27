BEGIN;

-- A GitHub authorization holds the user access token that a member grants to
-- the public SuperPlane GitHub App. SuperPlane uses it only to open pull
-- requests as that member. A member opts in from the account profile.
CREATE TABLE account_github_authorizations (
  account_id UUID PRIMARY KEY
    REFERENCES accounts(id) ON DELETE CASCADE,
  github_app_id BIGINT NOT NULL,
  github_user_id VARCHAR(255) NOT NULL,
  github_login VARCHAR(255) NOT NULL,
  encrypted_access_token BYTEA NOT NULL,
  access_token_expires_at TIMESTAMPTZ,
  encrypted_refresh_token BYTEA NOT NULL DEFAULT ''::bytea,
  refresh_token_expires_at TIMESTAMPTZ,
  invalidated_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT account_github_authorizations_user_present CHECK (btrim(github_user_id) <> ''),
  CONSTRAINT account_github_authorizations_login_present CHECK (btrim(github_login) <> '')
);

COMMIT;
