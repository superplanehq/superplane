BEGIN;

-- Remove ON DELETE CASCADE constraints from workflow-related tables
-- and recreate them without cascade to allow manual cleanup control

-- workflow_events table
ALTER TABLE workflow_events DROP CONSTRAINT workflow_events_workflow_id_fkey;
ALTER TABLE workflow_events ADD CONSTRAINT workflow_events_workflow_id_fkey
  FOREIGN KEY (workflow_id) REFERENCES workflows(id);

-- workflow_nodes table
ALTER TABLE workflow_nodes DROP CONSTRAINT workflow_nodes_workflow_id_fkey;
ALTER TABLE workflow_nodes ADD CONSTRAINT workflow_nodes_workflow_id_fkey
  FOREIGN KEY (workflow_id) REFERENCES workflows(id);

-- workflow_node_queue_items table
ALTER TABLE workflow_node_queue_items DROP CONSTRAINT workflow_node_queue_items_workflow_id_fkey;
ALTER TABLE workflow_node_queue_items ADD CONSTRAINT workflow_node_queue_items_workflow_id_fkey
  FOREIGN KEY (workflow_id) REFERENCES workflows(id);

ALTER TABLE workflow_node_queue_items DROP CONSTRAINT workflow_node_queue_items_root_event_id_fkey;
ALTER TABLE workflow_node_queue_items ADD CONSTRAINT workflow_node_queue_items_root_event_id_fkey
  FOREIGN KEY (root_event_id) REFERENCES workflow_events(id);

ALTER TABLE workflow_node_queue_items DROP CONSTRAINT workflow_node_queue_items_event_id_fkey;
ALTER TABLE workflow_node_queue_items ADD CONSTRAINT workflow_node_queue_items_event_id_fkey
  FOREIGN KEY (event_id) REFERENCES workflow_events(id);

-- workflow_node_executions table
ALTER TABLE workflow_node_executions DROP CONSTRAINT workflow_node_executions_workflow_id_fkey;
ALTER TABLE workflow_node_executions ADD CONSTRAINT workflow_node_executions_workflow_id_fkey
  FOREIGN KEY (workflow_id) REFERENCES workflows(id);

ALTER TABLE workflow_node_executions DROP CONSTRAINT workflow_node_executions_root_event_id_fkey;
ALTER TABLE workflow_node_executions ADD CONSTRAINT workflow_node_executions_root_event_id_fkey
  FOREIGN KEY (root_event_id) REFERENCES workflow_events(id);

ALTER TABLE workflow_node_executions DROP CONSTRAINT workflow_node_executions_event_id_fkey;
ALTER TABLE workflow_node_executions ADD CONSTRAINT workflow_node_executions_event_id_fkey
  FOREIGN KEY (event_id) REFERENCES workflow_events(id);

ALTER TABLE workflow_node_executions DROP CONSTRAINT workflow_node_executions_previous_execution_id_fkey;
ALTER TABLE workflow_node_executions ADD CONSTRAINT workflow_node_executions_previous_execution_id_fkey
  FOREIGN KEY (previous_execution_id) REFERENCES workflow_node_executions(id);

ALTER TABLE workflow_node_executions DROP CONSTRAINT workflow_node_executions_parent_execution_id_fkey;
ALTER TABLE workflow_node_executions ADD CONSTRAINT workflow_node_executions_parent_execution_id_fkey
  FOREIGN KEY (parent_execution_id) REFERENCES workflow_node_executions(id);

-- workflow_node_requests table (renamed from workflow_node_execution_requests)
ALTER TABLE workflow_node_requests DROP CONSTRAINT workflow_node_execution_requests_workflow_id_fkey;
ALTER TABLE workflow_node_requests ADD CONSTRAINT workflow_node_execution_requests_workflow_id_fkey
  FOREIGN KEY (workflow_id) REFERENCES workflows(id);

ALTER TABLE workflow_node_requests DROP CONSTRAINT workflow_node_execution_requests_execution_id_fkey;
ALTER TABLE workflow_node_requests ADD CONSTRAINT workflow_node_execution_requests_execution_id_fkey
  FOREIGN KEY (execution_id) REFERENCES workflow_node_executions(id);

-- workflow_node_execution_kvs table
ALTER TABLE workflow_node_execution_kvs DROP CONSTRAINT fk_wnek_workflow;
ALTER TABLE workflow_node_execution_kvs ADD CONSTRAINT fk_wnek_workflow
  FOREIGN KEY (workflow_id) REFERENCES workflows(id);

ALTER TABLE workflow_node_execution_kvs DROP CONSTRAINT fk_wnek_workflow_node;
ALTER TABLE workflow_node_execution_kvs ADD CONSTRAINT fk_wnek_workflow_node
  FOREIGN KEY (workflow_id, node_id) REFERENCES workflow_nodes(workflow_id, node_id);

ALTER TABLE workflow_node_execution_kvs DROP CONSTRAINT workflow_node_execution_kvs_execution_id_fkey;
ALTER TABLE workflow_node_execution_kvs ADD CONSTRAINT workflow_node_execution_kvs_execution_id_fkey
  FOREIGN KEY (execution_id) REFERENCES workflow_node_executions(id);

COMMIT;