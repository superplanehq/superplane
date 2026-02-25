DROP INDEX IF EXISTS idx_workflow_run_sessions_workflow_root;
DROP TABLE IF EXISTS workflow_run_sessions;

DROP INDEX IF EXISTS idx_workflow_events_workflow_continuation_key;
ALTER TABLE workflow_events DROP COLUMN IF EXISTS continuation_key;
