begin;

ALTER TABLE webhooks
  ADD COLUMN provisioning_mode TEXT NOT NULL DEFAULT 'legacy';

-- Lets the reconciler and ops provisioner efficiently find 'ops'-mode rows.
CREATE INDEX idx_webhooks_provisioning_mode
  ON webhooks(provisioning_mode)
  WHERE deleted_at IS NULL;

commit;
