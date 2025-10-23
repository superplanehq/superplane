BEGIN;

ALTER TABLE workflow_node_execution_requests RENAME TO workflow_node_requests;

ALTER TABLE workflow_node_requests
  ADD COLUMN node_id CHARACTER VARYING(128) NOT NULL,
  ADD FOREIGN KEY (workflow_id, node_id) REFERENCES workflow_nodes(workflow_id, node_id) ON DELETE CASCADE,
  ALTER column execution_id DROP NOT NULL;

ALTER INDEX idx_node_execution_requests_state_run_at RENAME TO idx_node_requests_state_run_at;

COMMIT;