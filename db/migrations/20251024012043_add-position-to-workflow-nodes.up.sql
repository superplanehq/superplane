BEGIN;

ALTER TABLE workflow_nodes ADD COLUMN position jsonb NOT NULL DEFAULT '{}';

COMMIT;