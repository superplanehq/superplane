BEGIN;

-- Add a nullable parent_node_id to support hierarchical workflow nodes.
ALTER TABLE workflow_nodes ADD COLUMN parent_node_id CHARACTER VARYING(128);

-- Create a composite foreign key referencing the same table (self-reference)
-- so that when a parent node is deleted, child nodes cascade as well.
ALTER TABLE workflow_nodes
  ADD CONSTRAINT fk_workflow_nodes_parent
  FOREIGN KEY (workflow_id, parent_node_id)
  REFERENCES workflow_nodes(workflow_id, node_id)
  ON DELETE CASCADE;

-- Index to speed up queries finding root/children by parent
CREATE INDEX IF NOT EXISTS idx_workflow_nodes_parent
  ON workflow_nodes(workflow_id, parent_node_id);

COMMIT;