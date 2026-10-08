BEGIN;

ALTER TABLE linear_webhook_receipts
  ADD COLUMN delivery_key TEXT NOT NULL DEFAULT '';

CREATE INDEX idx_linear_webhook_receipts_delivery
  ON linear_webhook_receipts (webhook_id, delivery_key)
  WHERE delivery_key <> '' AND outcome = 'accepted';

COMMIT;
