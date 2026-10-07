BEGIN;

--
-- Work-order-scoped checks. Scored assessments automations report on a
-- work order (risk review, coverage, confidence). One row per check key:
-- re-reporting the same key updates the row in place (latest-only state);
-- each report also lands an `order.check.reported` timeline event.
--
CREATE TABLE factory_work_order_checks (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  organization_id UUID NOT NULL,
  factory_id      UUID NOT NULL REFERENCES factories(id) ON DELETE RESTRICT,
  work_order_id   UUID NOT NULL REFERENCES factory_work_orders(id) ON DELETE RESTRICT,
  key             VARCHAR(255) NOT NULL,
  name            VARCHAR(255) NOT NULL,
  score           DOUBLE PRECISION NOT NULL,
  max_score       DOUBLE PRECISION NOT NULL,
  format          VARCHAR(16) NOT NULL DEFAULT 'fraction',
  level           VARCHAR(16) NOT NULL DEFAULT 'neutral',
  previous_score  DOUBLE PRECISION,
  summary         TEXT NOT NULL DEFAULT '',
  analysis        TEXT NOT NULL DEFAULT '',
  automation      JSONB,
  run_id          UUID,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_factory_work_order_checks_order_key_unique
  ON factory_work_order_checks (work_order_id, key);

CREATE INDEX idx_factory_work_order_checks_factory_created
  ON factory_work_order_checks (factory_id, created_at DESC);

COMMIT;
