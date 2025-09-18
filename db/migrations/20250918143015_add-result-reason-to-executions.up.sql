BEGIN;

ALTER TABLE stage_executions
  ADD COLUMN result_reason CHARACTER VARYING(64),
  ADD COLUMN result_message TEXT;

COMMIT;
