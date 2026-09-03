CREATE UNIQUE INDEX IF NOT EXISTS idx_workflow_node_execution_consumed_events_exec_event
    ON workflow_node_execution_consumed_events (execution_id, event_id);

DROP INDEX IF EXISTS idx_workflow_node_execution_consumed_events_execution_id;
