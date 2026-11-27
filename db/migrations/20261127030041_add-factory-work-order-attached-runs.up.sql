CREATE TABLE factory_work_order_attached_runs (
  work_order_id uuid NOT NULL REFERENCES factory_work_orders(id) ON DELETE CASCADE,
  run_id uuid NOT NULL REFERENCES workflow_runs(id) ON DELETE CASCADE,
  stage_key text NOT NULL,
  created_at timestamp with time zone NOT NULL DEFAULT now(),
  PRIMARY KEY (work_order_id, run_id)
);

CREATE INDEX idx_factory_work_order_attached_runs_run ON factory_work_order_attached_runs (run_id);
