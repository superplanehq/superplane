BEGIN;

-- The webhook provisioner polls for pending webhooks every second. Without an
-- access path for the state filter, Postgres scans every active webhook to find
-- the small pending set. This partial index holds only the pending rows, so the
-- idle poll cost tracks the number of pending webhooks rather than the total.
CREATE INDEX IF NOT EXISTS idx_webhooks_pending
  ON webhooks (created_at)
  WHERE state = 'pending' AND deleted_at IS NULL;

COMMIT;
