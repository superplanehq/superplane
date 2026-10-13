BEGIN;

ALTER TABLE factory_work_orders
  ADD COLUMN vcs_provider TEXT;

COMMIT;
