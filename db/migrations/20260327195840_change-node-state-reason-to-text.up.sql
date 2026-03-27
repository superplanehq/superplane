BEGIN;

ALTER TABLE workflow_nodes ALTER COLUMN state_reason TYPE text;
ALTER TABLE workflow_nodes ALTER COLUMN state_reason SET DEFAULT NULL::text;

COMMIT;
