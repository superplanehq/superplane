BEGIN;

ALTER TABLE stage_executions ADD COLUMN cancelled_at TIMESTAMP;
ALTER TABLE stage_executions ADD COLUMN cancelled_by UUID;

COMMIT;
