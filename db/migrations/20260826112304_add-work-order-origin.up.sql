BEGIN;

ALTER TABLE factory_work_orders
  ADD COLUMN origin_url TEXT,
  ADD COLUMN origin_label TEXT;

COMMIT;
