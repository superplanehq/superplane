BEGIN;

CREATE TABLE organizations (
    id                    UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name                  VARCHAR(255) NOT NULL UNIQUE,
    description           TEXT DEFAULT '',
    display_name          VARCHAR(255) NOT NULL,
    allowed_providers     JSONB NOT NULL DEFAULT '[]',
    email_domains_allowed JSONB DEFAULT '[]',
    created_at            TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at            TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at            TIMESTAMP
);

CREATE INDEX idx_organizations_deleted_at ON organizations(deleted_at);

CREATE TABLE organization_invitations (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  email           VARCHAR(255) NOT NULL,
  invited_by      UUID NOT NULL,
  status          VARCHAR(20) NOT NULL DEFAULT 'pending',
  expires_at      TIMESTAMP NOT NULL,
  created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT      unique_org_email UNIQUE(organization_id, email)
);

CREATE TABLE users (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  email           VARCHAR(255) NOT NULL,
  name            VARCHAR(255) NOT NULL,
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

  UNIQUE(organization_id, email)
);

CREATE TABLE account_providers (
  id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id          UUID NOT NULL REFERENCES users(id),
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
  
  UNIQUE(user_id, provider),
  UNIQUE(provider, provider_id)
);


COMMIT;
