BEGIN;

CREATE TABLE linear_webhook_receipts (
  id                 UUID PRIMARY KEY,
  received_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  integration_id     UUID NOT NULL,
  organization_id    UUID NOT NULL,
  webhook_id         UUID NOT NULL,
  event_type         TEXT NOT NULL DEFAULT '',
  action             TEXT NOT NULL DEFAULT '',
  issue_identifier   TEXT NOT NULL DEFAULT '',
  issue_id           TEXT NOT NULL DEFAULT '',
  team_key           TEXT NOT NULL DEFAULT '',
  workspace_key      TEXT NOT NULL DEFAULT '',
  http_status        INT NOT NULL,
  outcome            TEXT NOT NULL,
  subscription_count INT NOT NULL DEFAULT 0,
  task_ids           TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_linear_webhook_receipts_received_at
  ON linear_webhook_receipts (received_at DESC);

COMMIT;
