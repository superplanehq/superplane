BEGIN;

CREATE TABLE organizations (
  id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  name              VARCHAR(255) NOT NULL UNIQUE,
  display_name      VARCHAR(255) NOT NULL,
  allowed_providers JSONB NOT NULL DEFAULT '[]',
  created_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  deleted_at        TIMESTAMP
);

CREATE INDEX idx_organizations_deleted_at ON organizations(deleted_at);

ALTER TABLE CANVASES ADD COLUMN organization_id UUID REFERENCES organizations(id) NOT NULL;

CREATE TABLE organization_invitations (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  email           VARCHAR(255) NOT NULL,
  invited_by      UUID NOT NULL,
  state           VARCHAR(20) NOT NULL DEFAULT 'pending',
  created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE users ADD COLUMN organization_id UUID REFERENCES organizations(id) NOT NULL;
ALTER TABLE users DROP CONSTRAINT users_account_id_email_key;
ALTER TABLE users ADD CONSTRAINT unique_user_in_organization UNIQUE(organization_id, account_id, email);
ALTER TABLE canvases DROP CONSTRAINT canvases_name_key;
ALTER TABLE canvases ADD CONSTRAINT unique_canvas_in_organization UNIQUE(organization_id, name);

COMMIT;