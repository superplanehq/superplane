BEGIN;

--
-- Widen the work order lifecycle beyond open/closed to include draft and ready,
-- and allow `failed` as a third close result alongside completed/rejected.
-- New rows default to `draft`; existing rows keep their current state.
--
ALTER TABLE factory_work_orders
  ALTER COLUMN state SET DEFAULT 'draft';

COMMIT;
