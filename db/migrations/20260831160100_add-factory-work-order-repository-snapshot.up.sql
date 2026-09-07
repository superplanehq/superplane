BEGIN;

ALTER TABLE factory_work_orders
  ADD COLUMN repository TEXT,
  ADD COLUMN default_branch TEXT;

COMMIT;
