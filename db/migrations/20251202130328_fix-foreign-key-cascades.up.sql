BEGIN;

-- Drop existing constraints that need cascade updates
ALTER TABLE workflow_node_queue_items DROP CONSTRAINT workflow_node_queue_items_event_id_fkey;
ALTER TABLE workflow_node_queue_items DROP CONSTRAINT workflow_node_queue_items_root_event_id_fkey;

ALTER TABLE workflow_events DROP CONSTRAINT workflow_events_workflow_id_fkey;

ALTER TABLE workflow_node_executions DROP CONSTRAINT workflow_node_executions_event_id_fkey;
ALTER TABLE workflow_node_executions DROP CONSTRAINT workflow_node_executions_root_event_id_fkey;
ALTER TABLE workflow_node_executions DROP CONSTRAINT workflow_node_executions_parent_execution_id_fkey;
ALTER TABLE workflow_node_executions DROP CONSTRAINT workflow_node_executions_previous_execution_id_fkey;
ALTER TABLE workflow_node_executions DROP CONSTRAINT workflow_node_executions_workflow_id_fkey;

ALTER TABLE workflow_node_execution_kvs DROP CONSTRAINT workflow_node_execution_kvs_execution_id_fkey;
ALTER TABLE workflow_node_execution_kvs DROP CONSTRAINT fk_wnek_workflow;
ALTER TABLE workflow_node_execution_kvs DROP CONSTRAINT fk_wnek_workflow_node;

ALTER TABLE workflow_node_requests DROP CONSTRAINT workflow_node_execution_requests_execution_id_fkey;
ALTER TABLE workflow_node_requests DROP CONSTRAINT workflow_node_execution_requests_workflow_id_fkey;

-- Make columns nullable that need ON DELETE SET NULL
ALTER TABLE workflow_node_queue_items ALTER COLUMN event_id DROP NOT NULL;
ALTER TABLE workflow_node_queue_items ALTER COLUMN root_event_id DROP NOT NULL;
ALTER TABLE workflow_node_executions ALTER COLUMN root_event_id DROP NOT NULL;
ALTER TABLE workflow_node_executions ALTER COLUMN event_id DROP NOT NULL;

-- Re-create constraints with proper cascade operations

-- queue_items
ALTER TABLE workflow_node_queue_items
    ADD CONSTRAINT workflow_node_queue_items_event_id_fkey
    FOREIGN KEY (event_id) REFERENCES workflow_events(id) ON DELETE SET NULL;

ALTER TABLE workflow_node_queue_items
    ADD CONSTRAINT workflow_node_queue_items_root_event_id_fkey
    FOREIGN KEY (root_event_id) REFERENCES workflow_events(id) ON DELETE SET NULL;

-- events
ALTER TABLE workflow_events
    ADD CONSTRAINT workflow_events_workflow_id_fkey
    FOREIGN KEY (workflow_id) REFERENCES workflows(id);

-- Add missing execution_id FK constraint to workflow_events
ALTER TABLE workflow_events
    ADD CONSTRAINT workflow_events_execution_id_fkey
    FOREIGN KEY (execution_id) REFERENCES workflow_node_executions(id) ON DELETE CASCADE;

-- executions
ALTER TABLE workflow_node_executions
    ADD CONSTRAINT workflow_node_executions_workflow_id_fkey
    FOREIGN KEY (workflow_id) REFERENCES workflows(id);

ALTER TABLE workflow_node_executions
    ADD CONSTRAINT workflow_node_executions_root_event_id_fkey
    FOREIGN KEY (root_event_id) REFERENCES workflow_events(id) ON DELETE SET NULL;

ALTER TABLE workflow_node_executions
    ADD CONSTRAINT workflow_node_executions_event_id_fkey
    FOREIGN KEY (event_id) REFERENCES workflow_events(id) ON DELETE SET NULL;

ALTER TABLE workflow_node_executions
    ADD CONSTRAINT workflow_node_executions_parent_execution_id_fkey
    FOREIGN KEY (parent_execution_id) REFERENCES workflow_node_executions(id) ON DELETE CASCADE;

ALTER TABLE workflow_node_executions
    ADD CONSTRAINT workflow_node_executions_previous_execution_id_fkey
    FOREIGN KEY (previous_execution_id) REFERENCES workflow_node_executions(id) ON DELETE SET NULL;

-- execution_kvs
ALTER TABLE workflow_node_execution_kvs
    ADD CONSTRAINT fk_wnek_workflow
    FOREIGN KEY (workflow_id) REFERENCES workflows(id);

ALTER TABLE workflow_node_execution_kvs
    ADD CONSTRAINT fk_wnek_workflow_node
    FOREIGN KEY (workflow_id, node_id) REFERENCES workflow_nodes(workflow_id, node_id);

ALTER TABLE workflow_node_execution_kvs
    ADD CONSTRAINT workflow_node_execution_kvs_execution_id_fkey
    FOREIGN KEY (execution_id) REFERENCES workflow_node_executions(id) ON DELETE CASCADE;

-- node_requests
ALTER TABLE workflow_node_requests
    ADD CONSTRAINT workflow_node_requests_execution_id_fkey
    FOREIGN KEY (execution_id) REFERENCES workflow_node_executions(id) ON DELETE CASCADE;

ALTER TABLE workflow_node_requests
    ADD CONSTRAINT workflow_node_requests_workflow_id_fkey
    FOREIGN KEY (workflow_id) REFERENCES workflows(id);

COMMIT;