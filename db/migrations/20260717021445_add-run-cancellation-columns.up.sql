ALTER TABLE workflow_runs
    ADD COLUMN IF NOT EXISTS cancelled_at timestamp without time zone,
    ADD COLUMN IF NOT EXISTS cancelled_by uuid REFERENCES users(id);

CREATE INDEX IF NOT EXISTS idx_workflow_runs_cancelling ON workflow_runs (cancelled_at)
    WHERE state = 'cancelling';
