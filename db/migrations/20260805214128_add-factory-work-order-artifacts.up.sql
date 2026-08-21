BEGIN;

--
-- Work-order-scoped artifacts. Rich, typed outputs a work order accumulates
-- (pull requests, markdown notes, and — later — screenshots or recordings).
--
CREATE TABLE factory_work_order_artifacts (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  organization_id UUID NOT NULL,
  factory_id      UUID NOT NULL REFERENCES factories(id) ON DELETE RESTRICT,
  work_order_id   UUID NOT NULL REFERENCES factory_work_orders(id) ON DELETE RESTRICT,
  type            VARCHAR(32) NOT NULL,
  data            JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_by_id   UUID REFERENCES users(id) ON DELETE RESTRICT,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_factory_work_order_artifacts_work_order_created
  ON factory_work_order_artifacts (work_order_id, created_at DESC);

CREATE INDEX idx_factory_work_order_artifacts_factory_created
  ON factory_work_order_artifacts (factory_id, created_at DESC);

COMMIT;
