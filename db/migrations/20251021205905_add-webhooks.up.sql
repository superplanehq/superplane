begin;

CREATE TABLE webhooks (
  id             uuid NOT NULL DEFAULT uuid_generate_v4(),
  state          CHARACTER VARYING(32) NOT NULL,
  secret         BYTEA NOT NULL,
  configuration  jsonb NOT NULL DEFAULT '{}',
  metadata       jsonb NOT NULL DEFAULT '{}',
  integration_id uuid,
  resource       jsonb NOT NULL DEFAULT '{}',
  created_at     TIMESTAMP NOT NULL,
  updated_at     TIMESTAMP NOT NULL,
  deleted_at     TIMESTAMP,

  PRIMARY KEY (id),
  FOREIGN KEY (integration_id) REFERENCES integrations(id)
);

ALTER TABLE workflow_nodes
  ADD COLUMN webhook_id uuid,
  ADD FOREIGN KEY (webhook_id) REFERENCES webhooks(id);

CREATE INDEX idx_webhooks_deleted_at ON webhooks(deleted_at);

commit;
