BEGIN;

ALTER TABLE runner_tasks
  ADD COLUMN error_message TEXT,
  ADD COLUMN completion_hash TEXT;

COMMIT;
