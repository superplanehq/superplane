begin;

CREATE TABLE webhooks (
  id             uuid NOT NULL DEFAULT uuid_generate_v4(),
  state          CHARACTER VARYING(32) NOT NULL,
  secret         BYTEA NOT NULL,
  configuration  jsonb NOT NULL DEFAULT '{}',
  metadata       jsonb NOT NULL DEFAULT '{}',
  integration_id uuid,
  resource_type  CHARACTER VARYING(64) NOT NULL,
  resource_id    CHARACTER VARYING(128) NOT NULL,
  created_at     TIMESTAMP NOT NULL,
  updated_at     TIMESTAMP NOT NULL,

  PRIMARY KEY (id),
  FOREIGN KEY (integration_id) REFERENCES integrations(id)
);

ALTER TABLE workflow_nodes
  ADD COLUMN webhook_id uuid,
  ADD FOREIGN KEY (webhook_id) REFERENCES webhooks(id);

commit;
