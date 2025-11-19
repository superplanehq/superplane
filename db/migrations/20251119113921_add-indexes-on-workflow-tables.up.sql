begin;

CREATE INDEX idx_workflow_events_execution_id ON workflow_events(execution_id);
CREATE INDEX idx_workflow_events_state ON workflow_events(state);
CREATE INDEX idx_workflow_node_executions_state_created_at ON workflow_node_executions(state, created_at DESC);
CREATE INDEX idx_workflow_node_executions_parent_state ON workflow_node_executions(parent_execution_id, state);

commit;
