begin;

CREATE TABLE app_installation_subscriptions (
  id              uuid NOT NULL DEFAULT uuid_generate_v4(),
  installation_id uuid NOT NULL,
  workflow_id     uuid NOT NULL,
  node_id         CHARACTER VARYING(128) NOT NULL,
  configuration   jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at      TIMESTAMP NOT NULL,
  updated_at      TIMESTAMP NOT NULL,

  PRIMARY KEY (id),
  FOREIGN KEY (workflow_id) REFERENCES workflows(id) ON DELETE CASCADE,
  FOREIGN KEY (installation_id) REFERENCES app_installations(id) ON DELETE CASCADE,
  FOREIGN KEY (workflow_id, node_id) REFERENCES workflow_nodes(workflow_id, node_id) ON DELETE CASCADE
);

CREATE INDEX idx_app_installation_subscriptions_installation ON app_installation_subscriptions(installation_id);
CREATE INDEX idx_app_installation_subscriptions_workflow ON app_installation_subscriptions(workflow_id);
CREATE INDEX idx_app_installation_subscriptions_node ON app_installation_subscriptions(workflow_id, node_id);

commit;
