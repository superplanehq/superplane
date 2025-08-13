BEGIN;

CREATE TABLE accounts (
  id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  email      VARCHAR(255) NOT NULL,
  name       VARCHAR(255) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

  UNIQUE(email)
);

CREATE TABLE users (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  account_id      UUID NOT NULL,
  name            VARCHAR(255),
  email           VARCHAR(255),
  created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

  UNIQUE(account_id, email),
  FOREIGN KEY (account_id) REFERENCES accounts(id)
);

CREATE TABLE account_providers (
  id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  account_id       UUID NOT NULL,
  provider         VARCHAR(50) NOT NULL,
  provider_id      VARCHAR(255) NOT NULL,
  username         VARCHAR(255),
  email            VARCHAR(255),
  name             VARCHAR(255),
  avatar_url       TEXT,
  access_token     TEXT,
  refresh_token    TEXT,
  token_expires_at TIMESTAMP,
  created_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

  UNIQUE(account_id, provider),
  UNIQUE(provider, provider_id),
  FOREIGN KEY (account_id) REFERENCES accounts(id)
);

CREATE INDEX idx_account_providers_account_id ON account_providers(account_id);
CREATE INDEX idx_account_providers_provider ON account_providers(provider);

COMMIT;