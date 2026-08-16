BEGIN;

DROP INDEX IF EXISTS idx_factory_work_order_executions_line_dispatch;

ALTER TABLE factory_work_order_executions
  DROP COLUMN IF EXISTS line_dispatch_id;

DROP TABLE IF EXISTS factory_work_order_line_dispatches;

COMMIT;
