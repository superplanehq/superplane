BEGIN;

DELETE FROM workflow_nodes
WHERE workflow_id IN (
  SELECT id FROM workflows WHERE is_template = true
);

DELETE FROM workflows
WHERE is_template = true;

DROP INDEX IF EXISTS idx_workflows_is_template;

ALTER TABLE workflows
  DROP COLUMN IF EXISTS is_template;

COMMIT;
