BEGIN;

ALTER TABLE factory_work_orders
  ADD COLUMN source_run_id UUID REFERENCES workflow_runs(id) ON DELETE SET NULL;

CREATE INDEX idx_factory_work_orders_source_run_id
  ON factory_work_orders (source_run_id)
  WHERE source_run_id IS NOT NULL;

COMMIT;
