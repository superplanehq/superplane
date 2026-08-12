BEGIN;

ALTER TABLE factory_work_order_executions
  ADD COLUMN total_tokens BIGINT NOT NULL DEFAULT 0,
  ADD COLUMN cost_cents BIGINT NOT NULL DEFAULT 0;

COMMIT;
