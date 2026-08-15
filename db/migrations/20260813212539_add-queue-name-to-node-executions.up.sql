--
-- The resolved queue name the execution occupies a slot in.
-- Capacity for node queues is derived by counting non-terminal
-- executions per (workflow_id, queue_name), backed by this partial index.
--
ALTER TABLE workflow_node_executions ADD COLUMN queue_name character varying(256);

CREATE INDEX idx_workflow_node_executions_active_queue
    ON workflow_node_executions (workflow_id, queue_name)
    WHERE state IN ('pending', 'started', 'cancelling');
