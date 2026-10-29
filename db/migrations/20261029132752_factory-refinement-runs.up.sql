CREATE TABLE factory_work_order_refinement_runs (
    run_id uuid PRIMARY KEY REFERENCES workflow_runs(id) ON DELETE CASCADE,
    work_order_id uuid NOT NULL REFERENCES factory_work_orders(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_factory_work_order_refinement_runs_work_order
    ON factory_work_order_refinement_runs (work_order_id);

INSERT INTO factory_work_order_refinement_runs (run_id, work_order_id)
SELECT canvas_run_id, draft_work_order_id
FROM factory_planning_sessions
WHERE canvas_run_id IS NOT NULL AND draft_work_order_id IS NOT NULL
ON CONFLICT DO NOTHING;
