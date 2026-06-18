begin;

DELETE FROM workflow_node_executions WHERE parent_execution_id IS NOT NULL;

ALTER TABLE workflow_node_executions DROP CONSTRAINT IF EXISTS workflow_node_executions_parent_execution_id_fkey;
DROP INDEX IF EXISTS idx_workflow_node_executions_parent_execution_id;
DROP INDEX IF EXISTS idx_workflow_node_executions_parent_state;
ALTER TABLE workflow_node_executions DROP COLUMN IF EXISTS parent_execution_id;

DELETE FROM workflow_nodes WHERE parent_node_id IS NOT NULL;
DELETE FROM workflow_nodes WHERE type = 'blueprint';

ALTER TABLE workflow_nodes DROP CONSTRAINT IF EXISTS fk_workflow_nodes_parent;
DROP INDEX IF EXISTS idx_workflow_nodes_parent;
ALTER TABLE workflow_nodes DROP COLUMN IF EXISTS parent_node_id;

DROP TABLE IF EXISTS blueprints;

DELETE FROM casbin_rule WHERE v2 = 'blueprints';

commit;
