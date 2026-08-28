BEGIN;

CREATE TABLE factory_work_order_surveys (
  id                   UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  organization_id      UUID NOT NULL,
  factory_id           UUID NOT NULL REFERENCES factories(id) ON DELETE RESTRICT,
  work_order_id        UUID NOT NULL REFERENCES factory_work_orders(id) ON DELETE RESTRICT,
  canvas_run_id        UUID NOT NULL,
  execution_id         UUID,
  status               TEXT NOT NULL,
  questions            JSONB NOT NULL DEFAULT '[]'::jsonb,
  answers              JSONB NOT NULL DEFAULT '[]'::jsonb,
  timeout_seconds      INTEGER NOT NULL,
  expires_at           TIMESTAMPTZ NOT NULL,
  created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  answered_at          TIMESTAMPTZ,
  answered_by_user_id  UUID REFERENCES users(id) ON DELETE RESTRICT
);

CREATE UNIQUE INDEX idx_factory_work_order_surveys_one_pending
  ON factory_work_order_surveys (work_order_id)
  WHERE status = 'pending';

CREATE INDEX idx_factory_work_order_surveys_work_order_created
  ON factory_work_order_surveys (work_order_id, created_at DESC);

COMMIT;
