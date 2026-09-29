BEGIN;

CREATE TABLE sentry_webhook_receipts (
  id                 UUID PRIMARY KEY,
  received_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  hook_resource      TEXT NOT NULL DEFAULT '',
  action             TEXT NOT NULL DEFAULT '',
  installation_uuid  TEXT NOT NULL DEFAULT '',
  organization_slug  TEXT NOT NULL DEFAULT '',
  project_slug       TEXT NOT NULL DEFAULT '',
  issue_id           TEXT NOT NULL DEFAULT '',
  issue_short_id     TEXT NOT NULL DEFAULT '',
  http_status        INT NOT NULL,
  outcome            TEXT NOT NULL,
  integration_count  INT NOT NULL DEFAULT 0
);

CREATE INDEX idx_sentry_webhook_receipts_received_at
  ON sentry_webhook_receipts (received_at DESC);

COMMIT;
