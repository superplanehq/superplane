BEGIN;

-- workflow_node_executions table
CREATE INDEX idx_workflow_node_executions_previous_execution_id 
ON workflow_node_executions(previous_execution_id);

CREATE INDEX idx_workflow_node_executions_parent_execution_id 
ON workflow_node_executions(parent_execution_id);

CREATE INDEX idx_workflow_node_executions_root_event_id 
ON workflow_node_executions(root_event_id);

CREATE INDEX idx_workflow_node_executions_event_id 
ON workflow_node_executions(event_id);

-- workflow_node_requests table
CREATE INDEX idx_workflow_node_requests_execution_id 
ON workflow_node_requests(execution_id);

-- workflow_node_queue_items table
CREATE INDEX idx_workflow_node_queue_items_root_event_id 
ON workflow_node_queue_items(root_event_id);

COMMIT;