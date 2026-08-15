--
-- Line-step admission control: a work order that becomes ready for a step
-- at capacity waits in the step's queue. Waiting is stored as a
-- factory_work_order_executions row with status 'waiting' and no run yet,
-- so run_id must be nullable.
--
ALTER TABLE factory_work_order_executions ALTER COLUMN run_id DROP NOT NULL;
