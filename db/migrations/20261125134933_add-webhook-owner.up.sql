BEGIN;

ALTER TABLE webhooks
  ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
  ADD COLUMN IF NOT EXISTS factory_id UUID REFERENCES factories(id) ON DELETE CASCADE,
  ADD COLUMN IF NOT EXISTS name TEXT;

CREATE INDEX IF NOT EXISTS idx_webhooks_factory_id ON webhooks (factory_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_webhooks_factory_name
  ON webhooks (factory_id, name)
  WHERE factory_id IS NOT NULL AND deleted_at IS NULL;

COMMIT;
