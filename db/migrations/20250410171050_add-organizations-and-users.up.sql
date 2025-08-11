BEGIN;

CREATE TABLE organizations (
  id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  name              VARCHAR(255) NOT NULL UNIQUE,
  description       TEXT DEFAULT '',
  display_name      VARCHAR(255) NOT NULL,
  allowed_providers JSONB NOT NULL DEFAULT '[]',
  created_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  deleted_at        TIMESTAMP
);

CREATE INDEX idx_organizations_deleted_at ON organizations(deleted_at);

CREATE TABLE organization_invitations (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  email           VARCHAR(255) NOT NULL,
  invited_by      UUID NOT NULL,
  status          VARCHAR(20) NOT NULL DEFAULT 'pending',
  created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

  UNIQUE(organization_id, email)
);

CREATE TABLE accounts (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  email           VARCHAR(255) NOT NULL,
  name            VARCHAR(255) NOT NULL,
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

  UNIQUE(email)
);

CREATE TABLE users (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  organization_id UUID NOT NULL,
  account_id      UUID NOT NULL,
  email           VARCHAR(255) NOT NULL,
  name            VARCHAR(255) NOT NULL,
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

  UNIQUE(organization_id, email),
  FOREIGN KEY (organization_id) REFERENCES organizations(id),
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


COMMIT;
