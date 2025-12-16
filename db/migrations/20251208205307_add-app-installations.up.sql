begin;

CREATE TABLE app_installations (
  id                uuid NOT NULL DEFAULT uuid_generate_v4(),
  organization_id   uuid NOT NULL,
  app_name          CHARACTER VARYING(255) NOT NULL,
  installation_name CHARACTER VARYING(255) NOT NULL,
  state             CHARACTER VARYING(32) NOT NULL,
  state_description CHARACTER VARYING(255),
  configuration     JSONB NOT NULL DEFAULT '{}',
  metadata          JSONB NOT NULL DEFAULT '{}',
  browser_action    JSONB,
  created_at        TIMESTAMP NOT NULL,
  updated_at        TIMESTAMP NOT NULL,

  PRIMARY KEY (id),
  FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE
);

CREATE INDEX idx_app_installations_organization_id ON app_installations(organization_id);

CREATE TABLE app_installation_secrets (
  id              uuid NOT NULL DEFAULT uuid_generate_v4(),
  organization_id uuid NOT NULL,
  installation_id uuid NOT NULL,
  name            CHARACTER VARYING(64) NOT NULL,
  value           BYTEA NOT NULL,
  created_at      TIMESTAMP NOT NULL,
  updated_at      TIMESTAMP NOT NULL,

  PRIMARY KEY (id),
  FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
  FOREIGN KEY (installation_id) REFERENCES app_installations(id) ON DELETE CASCADE
);

CREATE INDEX idx_app_installation_secrets_organization_id ON app_installation_secrets(organization_id);
CREATE INDEX idx_app_installation_secrets_installation_id ON app_installation_secrets(installation_id);

--
-- Add app_installation_id to workflow_nodes
--

ALTER TABLE workflow_nodes
  ADD COLUMN app_installation_id uuid,
  ADD FOREIGN KEY (app_installation_id) REFERENCES app_installations(id) ON DELETE SET NULL;

CREATE INDEX idx_workflow_node_installation_id ON workflow_nodes(app_installation_id);

--
-- Add app_installation_id to webhooks
--

ALTER TABLE webhooks
  ADD COLUMN app_installation_id uuid,
  ADD FOREIGN KEY (app_installation_id) REFERENCES app_installations(id);

CREATE INDEX idx_webhooks_app_installation_id ON webhooks(app_installation_id);

commit;
