BEGIN;

CREATE TABLE organizations (
  id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  name              VARCHAR(255) NOT NULL UNIQUE,
  display_name      VARCHAR(255) NOT NULL,
  allowed_providers JSONB NOT NULL DEFAULT '[]',
  created_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  deleted_at        TIMESTAMP,

  UNIQUE(name)
);

CREATE TABLE organization_invitations (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  email           VARCHAR(255) NOT NULL,
  invited_by      UUID NOT NULL,
  status          VARCHAR(20) NOT NULL,
  expires_at      TIMESTAMP NOT NULL,
  created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

  UNIQUE(organization_id, email)
);

CREATE INDEX idx_organizations_deleted_at ON organizations(deleted_at);

ALTER TABLE CANVASES ADD COLUMN organization_id UUID REFERENCES organizations(id) NOT NULL;

COMMIT;
