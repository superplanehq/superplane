BEGIN;

-- Remove the self-referencing foreign key and column
ALTER TABLE workflow_nodes DROP CONSTRAINT IF EXISTS fk_workflow_nodes_parent;
DROP INDEX IF EXISTS idx_workflow_nodes_parent;
ALTER TABLE workflow_nodes DROP COLUMN IF EXISTS parent_node_id;

COMMIT;