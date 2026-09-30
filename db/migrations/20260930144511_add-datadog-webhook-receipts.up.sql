BEGIN;

CREATE TABLE datadog_webhook_receipts (
  id                  UUID PRIMARY KEY,
  received_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  integration_id      UUID NOT NULL,
  organization_id     UUID NOT NULL,
  event_type          TEXT NOT NULL DEFAULT '',
  alert_transition    TEXT NOT NULL DEFAULT '',
  alert_id            TEXT NOT NULL DEFAULT '',
  service             TEXT NOT NULL DEFAULT '',
  issue_id            TEXT NOT NULL DEFAULT '',
  http_status         INT NOT NULL,
  outcome             TEXT NOT NULL,
  subscription_count  INT NOT NULL DEFAULT 0,
  task_ids            TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_datadog_webhook_receipts_received_at
  ON datadog_webhook_receipts (received_at DESC);

COMMIT;
