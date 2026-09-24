BEGIN;

CREATE TABLE factory_agent_resources (
  id                   UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  organization_id      UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
  factory_id           UUID NOT NULL REFERENCES factories(id) ON DELETE RESTRICT,
  kind                 TEXT NOT NULL,
  name                 TEXT NOT NULL,
  enabled              BOOLEAN NOT NULL DEFAULT TRUE,
  config               JSONB NOT NULL DEFAULT '{}'::jsonb,
  oauth_status         TEXT NOT NULL DEFAULT '',
  oauth_error          TEXT NOT NULL DEFAULT '',
  oauth_connected_by   UUID REFERENCES users(id) ON DELETE SET NULL,
  oauth_connected_at   TIMESTAMPTZ,
  oauth_metadata       JSONB NOT NULL DEFAULT '{}'::jsonb,
  oauth_pending_state  TEXT NOT NULL DEFAULT '',
  oauth_pending_expiry TIMESTAMPTZ,
  created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_factory_agent_resources_factory_name
  ON factory_agent_resources (factory_id, name);

CREATE INDEX idx_factory_agent_resources_factory_kind
  ON factory_agent_resources (factory_id, kind);

CREATE UNIQUE INDEX idx_factory_agent_resources_oauth_pending_state
  ON factory_agent_resources (oauth_pending_state)
  WHERE oauth_pending_state <> '';

CREATE TABLE factory_agent_resource_secrets (
  id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  resource_id UUID NOT NULL REFERENCES factory_agent_resources(id) ON DELETE CASCADE,
  name        TEXT NOT NULL,
  value       BYTEA NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_factory_agent_resource_secrets_resource_name
  ON factory_agent_resource_secrets (resource_id, name);

COMMIT;
