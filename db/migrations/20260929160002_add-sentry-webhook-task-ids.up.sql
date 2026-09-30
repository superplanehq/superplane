BEGIN;

ALTER TABLE sentry_webhook_receipts
  ADD COLUMN task_ids TEXT NOT NULL DEFAULT '';

COMMIT;
