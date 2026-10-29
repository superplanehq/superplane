CREATE TABLE factory_work_order_refinement_runs (
    run_id uuid PRIMARY KEY REFERENCES workflow_runs(id) ON DELETE CASCADE,
    work_order_id uuid NOT NULL REFERENCES factory_work_orders(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_factory_work_order_refinement_runs_work_order
    ON factory_work_order_refinement_runs (work_order_id);

INSERT INTO factory_work_order_refinement_runs (run_id, work_order_id)
SELECT s.canvas_run_id, s.draft_work_order_id
FROM factory_planning_sessions s
JOIN workflow_runs r ON r.id = s.canvas_run_id
WHERE s.draft_work_order_id IS NOT NULL
ON CONFLICT DO NOTHING;
