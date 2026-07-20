BEGIN;

-- Speeds orphan sweeps / existence checks by (workflow_id, node_id), and FK
-- checks when hard-deleting workflow_nodes.
CREATE INDEX idx_workflow_node_requests_workflow_node_id
  ON workflow_node_requests (workflow_id, node_id);

CREATE INDEX idx_workflow_node_queue_items_workflow_node_id
  ON workflow_node_queue_items (workflow_id, node_id);

COMMIT;
