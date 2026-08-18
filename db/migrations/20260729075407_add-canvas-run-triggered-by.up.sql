ALTER TABLE canvas_runs
    ADD COLUMN IF NOT EXISTS triggered_by uuid REFERENCES users(id);

CREATE INDEX IF NOT EXISTS idx_canvas_runs_workflow_triggered_by_created_at
    ON canvas_runs (workflow_id, triggered_by, created_at DESC)
    WHERE triggered_by IS NOT NULL;
