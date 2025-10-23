BEGIN;

ALTER TABLE workflow_nodes ADD COLUMN metadata jsonb NOT NULL DEFAULT '{}';

COMMIT;