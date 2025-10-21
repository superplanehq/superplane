begin;

CREATE TABLE webhooks (
  id     uuid NOT NULL DEFAULT uuid_generate_v4(),
  secret BYTEA NOT NULL,

  PRIMARY KEY (id)
);

CREATE TABLE webhook_handlers (
  id          uuid NOT NULL DEFAULT uuid_generate_v4(),
  webhook_id  uuid NOT NULL,
  workflow_id uuid NOT NULL,
  node_id     CHARACTER VARYING(128) NOT NULL,
  spec        JSONB NOT NULL,

  PRIMARY KEY (id),
  FOREIGN KEY (webhook_id) REFERENCES webhooks(id) ON DELETE CASCADE,
  FOREIGN KEY (workflow_id, node_id) REFERENCES workflow_nodes(workflow_id, node_id) ON DELETE CASCADE
);

commit;
