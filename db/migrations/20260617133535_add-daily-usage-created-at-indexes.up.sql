BEGIN;

CREATE INDEX IF NOT EXISTS idx_workflow_runs_created_at ON workflow_runs(created_at);

CREATE INDEX IF NOT EXISTS idx_workflow_events_created_at ON workflow_events(created_at);

CREATE INDEX IF NOT EXISTS idx_workflow_node_executions_created_at ON workflow_node_executions(created_at);

COMMIT;
