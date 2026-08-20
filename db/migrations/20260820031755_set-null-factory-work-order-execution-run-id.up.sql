BEGIN;

ALTER TABLE factory_work_order_executions
  ALTER COLUMN run_id DROP NOT NULL;

ALTER TABLE factory_work_order_executions
  DROP CONSTRAINT factory_work_order_executions_run_id_fkey;

ALTER TABLE factory_work_order_executions
  ADD CONSTRAINT factory_work_order_executions_run_id_fkey
  FOREIGN KEY (run_id) REFERENCES workflow_runs(id) ON DELETE SET NULL;

COMMIT;
