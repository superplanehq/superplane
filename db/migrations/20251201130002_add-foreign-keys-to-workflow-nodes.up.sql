BEGIN;

-- Add foreign keys for tables that reference workflow_nodes (workflow_id, node_id)
ALTER TABLE workflow_events ADD CONSTRAINT fk_workflow_events_workflow_node
  FOREIGN KEY (workflow_id, node_id) REFERENCES workflow_nodes(workflow_id, node_id);

ALTER TABLE workflow_node_executions ADD CONSTRAINT fk_workflow_node_executions_workflow_node
  FOREIGN KEY (workflow_id, node_id) REFERENCES workflow_nodes(workflow_id, node_id);

ALTER TABLE workflow_node_queue_items ADD CONSTRAINT fk_workflow_node_queue_items_workflow_node
  FOREIGN KEY (workflow_id, node_id) REFERENCES workflow_nodes(workflow_id, node_id);

ALTER TABLE workflow_node_requests ADD CONSTRAINT fk_workflow_node_requests_workflow_node
  FOREIGN KEY (workflow_id, node_id) REFERENCES workflow_nodes(workflow_id, node_id);

COMMIT;