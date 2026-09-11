BEGIN;

CREATE INDEX idx_factory_work_orders_factory_origin_url
  ON factory_work_orders (factory_id, origin_url)
  WHERE origin_url IS NOT NULL;

COMMIT;
