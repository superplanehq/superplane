BEGIN;

ALTER TABLE workflow_node_executions ADD COLUMN cancelled_by UUID;

COMMIT;
