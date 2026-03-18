BEGIN;

CREATE TABLE usage_profiles (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  name VARCHAR(255) NOT NULL UNIQUE,
  max_orgs_per_account INTEGER NOT NULL DEFAULT 0,
  max_canvases_per_org INTEGER NOT NULL DEFAULT 0,
  max_nodes_per_canvas INTEGER NOT NULL DEFAULT 0,
  max_users_per_org INTEGER NOT NULL DEFAULT 0,
  max_integrations_per_org INTEGER NOT NULL DEFAULT 0,
  max_events_per_month INTEGER NOT NULL DEFAULT 0,
  retention_days INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE org_usage_overrides (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  organization_id UUID NOT NULL REFERENCES organizations(id),
  max_orgs_per_account INTEGER,
  max_canvases_per_org INTEGER,
  max_nodes_per_canvas INTEGER,
  max_users_per_org INTEGER,
  max_integrations_per_org INTEGER,
  max_events_per_month INTEGER,
  retention_days INTEGER,
  is_unlimited BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_org_usage_overrides_organization_id ON org_usage_overrides(organization_id);

ALTER TABLE organizations ADD COLUMN usage_profile_id UUID REFERENCES usage_profiles(id);

INSERT INTO usage_profiles (name, max_orgs_per_account, max_canvases_per_org, max_nodes_per_canvas, max_users_per_org, max_integrations_per_org, max_events_per_month, retention_days)
VALUES ('basic', 1, 3, 50, 3, 5, 10000, 14);

COMMIT;
