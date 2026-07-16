BEGIN;

ALTER TABLE workflow_runs ADD COLUMN cancelled_at TIMESTAMP;
ALTER TABLE workflow_runs ADD COLUMN cancelled_by UUID;

COMMIT;
