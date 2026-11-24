BEGIN;

ALTER TABLE webhooks
  ADD COLUMN organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
  ADD COLUMN factory_id UUID REFERENCES factories(id) ON DELETE CASCADE,
  ADD COLUMN name TEXT;

CREATE INDEX idx_webhooks_factory_id ON webhooks (factory_id);

CREATE UNIQUE INDEX idx_webhooks_factory_name
  ON webhooks (factory_id, name)
  WHERE factory_id IS NOT NULL AND deleted_at IS NULL;

COMMIT;
