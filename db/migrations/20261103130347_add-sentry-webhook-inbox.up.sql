CREATE TABLE sentry_webhook_inbox (
  id uuid PRIMARY KEY,
  receipt_id uuid REFERENCES sentry_webhook_receipts(id) ON DELETE SET NULL,
  source text NOT NULL,
  integration_id uuid,
  installation_uuid text NOT NULL DEFAULT '',
  hook_resource text NOT NULL DEFAULT '',
  action text NOT NULL DEFAULT '',
  priority integer NOT NULL DEFAULT 0,
  payload_ciphertext bytea,
  state text NOT NULL,
  targets_resolved boolean NOT NULL DEFAULT false,
  attempts integer NOT NULL DEFAULT 0,
  next_attempt_at timestamp with time zone NOT NULL,
  lease_owner uuid,
  lease_expires_at timestamp with time zone,
  last_error text NOT NULL DEFAULT '',
  received_at timestamp with time zone NOT NULL,
  completed_at timestamp with time zone,
  updated_at timestamp with time zone NOT NULL
);

CREATE INDEX idx_sentry_webhook_inbox_due
  ON sentry_webhook_inbox (priority DESC, next_attempt_at, received_at)
  WHERE completed_at IS NULL;

CREATE INDEX idx_sentry_webhook_inbox_installation
  ON sentry_webhook_inbox (installation_uuid)
  WHERE completed_at IS NULL AND hook_resource = 'installation';

CREATE INDEX idx_sentry_webhook_inbox_completed_at
  ON sentry_webhook_inbox (completed_at)
  WHERE completed_at IS NOT NULL;

CREATE INDEX idx_sentry_webhook_inbox_receipt_id
  ON sentry_webhook_inbox (receipt_id);

CREATE TABLE sentry_webhook_delivery_targets (
  id uuid PRIMARY KEY,
  inbox_id uuid NOT NULL REFERENCES sentry_webhook_inbox(id) ON DELETE CASCADE,
  integration_id uuid NOT NULL,
  workflow_id uuid NOT NULL,
  node_id text NOT NULL,
  state text NOT NULL,
  last_error text NOT NULL DEFAULT '',
  completed_at timestamp with time zone,
  created_at timestamp with time zone NOT NULL,
  updated_at timestamp with time zone NOT NULL,
  CONSTRAINT uq_sentry_webhook_delivery_targets_identity UNIQUE (inbox_id, integration_id, workflow_id, node_id)
);

CREATE TABLE sentry_app_installation_deletions (
  installation_uuid text PRIMARY KEY,
  deleted_at timestamp with time zone NOT NULL
);

ALTER TABLE sentry_webhook_receipts
  ADD COLUMN processing_error text NOT NULL DEFAULT '';
