begin;

CREATE TABLE webhook_subscription_bindings (
  id                  uuid        NOT NULL DEFAULT uuid_generate_v4(),
  organization_id     uuid        NOT NULL,
  app_installation_id uuid        NOT NULL,
  workflow_id         uuid        NOT NULL,
  node_id             TEXT        NOT NULL,
  webhook_id          uuid,
  scope_key           TEXT        NOT NULL,
  requested_config    JSONB       NOT NULL DEFAULT '{}',
  requested_hash      TEXT        NOT NULL,
  active              BOOLEAN     NOT NULL DEFAULT true,
  created_at          TIMESTAMP   NOT NULL,
  updated_at          TIMESTAMP   NOT NULL,
  deleted_at          TIMESTAMP,

  PRIMARY KEY (id),
  FOREIGN KEY (organization_id)     REFERENCES organizations(id)      ON DELETE CASCADE,
  FOREIGN KEY (app_installation_id) REFERENCES app_installations(id)  ON DELETE CASCADE,
  FOREIGN KEY (workflow_id)         REFERENCES workflows(id)          ON DELETE CASCADE,
  FOREIGN KEY (webhook_id)          REFERENCES webhooks(id)           ON DELETE SET NULL
);

-- Reconciler query: find all active bindings for a given installation + scope group.
CREATE INDEX idx_bindings_install_scope_active
  ON webhook_subscription_bindings(app_installation_id, scope_key)
  WHERE active = true AND deleted_at IS NULL;

-- Lookup by canvas node for upsert during RequestWebhook.
CREATE INDEX idx_bindings_node
  ON webhook_subscription_bindings(workflow_id, node_id)
  WHERE deleted_at IS NULL;

-- One active binding per node at a time.
CREATE UNIQUE INDEX idx_bindings_node_unique_active
  ON webhook_subscription_bindings(workflow_id, node_id)
  WHERE active = true AND deleted_at IS NULL;

commit;
