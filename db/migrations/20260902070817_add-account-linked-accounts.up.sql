BEGIN;

-- A linked account records an identity that a member owns on another service.
-- It is not a sign-in method: it grants no session and carries no token.
-- SuperPlane uses it to attribute activity, such as repository authorship in
-- Velocity reports.
CREATE TABLE account_linked_accounts (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  account_id UUID NOT NULL
    REFERENCES accounts(id) ON DELETE CASCADE,
  provider VARCHAR(50) NOT NULL,
  provider_id VARCHAR(255) NOT NULL,
  username VARCHAR(255) NOT NULL,
  name VARCHAR(255),
  avatar_url TEXT,
  linked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT account_linked_accounts_provider_present CHECK (btrim(provider) <> ''),
  CONSTRAINT account_linked_accounts_provider_id_present CHECK (btrim(provider_id) <> ''),
  CONSTRAINT account_linked_accounts_username_present CHECK (btrim(username) <> '')
);

-- One identity per service per account.
CREATE UNIQUE INDEX idx_account_linked_accounts_account_provider
  ON account_linked_accounts (account_id, provider);

-- One account per identity, so two members cannot claim the same author.
CREATE UNIQUE INDEX idx_account_linked_accounts_provider_identity
  ON account_linked_accounts (provider, provider_id);

-- Velocity resolves a member by the lowercased login.
CREATE INDEX idx_account_linked_accounts_provider_username
  ON account_linked_accounts (provider, lower(username));

COMMIT;
