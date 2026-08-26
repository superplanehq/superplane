BEGIN;

ALTER TABLE factory_work_orders
  ADD COLUMN origin_url TEXT,
  ADD COLUMN origin_label TEXT;

CREATE UNIQUE INDEX idx_factory_work_orders_factory_origin_url
  ON factory_work_orders (factory_id, origin_url)
  WHERE origin_url IS NOT NULL AND origin_url <> '';

COMMIT;
