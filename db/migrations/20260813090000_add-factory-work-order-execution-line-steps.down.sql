BEGIN;

ALTER TABLE factory_work_order_executions
  DROP COLUMN line_steps;

COMMIT;
