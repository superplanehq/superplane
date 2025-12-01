BEGIN;

-- Add soft deletion column to workflow_nodes
ALTER TABLE workflow_nodes ADD COLUMN deleted_at timestamp with time zone;
CREATE INDEX idx_workflow_nodes_deleted_at ON workflow_nodes (deleted_at);

COMMIT;