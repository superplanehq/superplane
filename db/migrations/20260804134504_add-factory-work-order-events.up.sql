CREATE TABLE factory_work_order_events (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  work_order_id   UUID NOT NULL REFERENCES factory_work_orders(id) ON DELETE RESTRICT,
  type            TEXT NOT NULL,
  data            JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_factory_work_order_events_work_order_created
  ON factory_work_order_events (work_order_id, created_at DESC);
