begin;

CREATE TABLE webhook_operations (
  id                  uuid        NOT NULL DEFAULT uuid_generate_v4(),
  webhook_id          uuid        NOT NULL,
  operation_type      TEXT        NOT NULL,
  desired_config      JSONB,
  desired_config_hash TEXT,
  idempotency_key     TEXT        NOT NULL,
  state               TEXT        NOT NULL,
  attempt_count       INTEGER     NOT NULL DEFAULT 0,
  max_attempts        INTEGER     NOT NULL DEFAULT 5,
  next_attempt_at     TIMESTAMP   NOT NULL DEFAULT NOW(),
  last_error_message  TEXT,
  last_error_at       TIMESTAMP,
  created_at          TIMESTAMP   NOT NULL,
  updated_at          TIMESTAMP   NOT NULL,

  PRIMARY KEY (id),
  FOREIGN KEY (webhook_id) REFERENCES webhooks(id) ON DELETE CASCADE,
  UNIQUE (idempotency_key)
);

-- Provisioner poll: queued ops whose next attempt time has arrived.
CREATE INDEX idx_webhook_ops_poll
  ON webhook_operations(state, next_attempt_at);

commit;
