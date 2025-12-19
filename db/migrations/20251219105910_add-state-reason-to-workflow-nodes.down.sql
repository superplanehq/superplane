BEGIN;

ALTER TABLE workflow_nodes DROP COLUMN state_reason;

COMMIT;