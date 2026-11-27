CREATE INDEX idx_factory_work_orders_origin_url ON factory_work_orders USING hash (origin_url) WHERE origin_url IS NOT NULL;
