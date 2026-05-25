begin;

ALTER TABLE webhooks
  ADD COLUMN scope_key            TEXT,
  ADD COLUMN config_hash          TEXT,
  ADD COLUMN provider_webhook_id  TEXT,
  ADD COLUMN provider_etag        TEXT,
  ADD COLUMN secret_version       INTEGER NOT NULL DEFAULT 1,
  ADD COLUMN last_provisioned_at  TIMESTAMP,
  ADD COLUMN last_error_code      TEXT,
  ADD COLUMN last_error_message   TEXT,
  ADD COLUMN last_error_at        TIMESTAMP;

CREATE INDEX idx_webhooks_app_installation_scope
  ON webhooks(app_installation_id, scope_key)
  WHERE deleted_at IS NULL;

commit;
