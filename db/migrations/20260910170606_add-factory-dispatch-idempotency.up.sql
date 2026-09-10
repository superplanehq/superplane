CREATE TABLE factory_work_order_dispatch_requests (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  organization_id UUID NOT NULL,
  factory_id UUID NOT NULL,
  work_order_id UUID NOT NULL,
  idempotency_key VARCHAR(255) NOT NULL,
  request_fingerprint VARCHAR(64) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT factory_work_order_dispatch_requests_work_order_key UNIQUE (work_order_id, idempotency_key),
  CONSTRAINT factory_work_order_dispatch_requests_factory_id_fkey
    FOREIGN KEY (factory_id) REFERENCES factories(id) ON DELETE RESTRICT,
  CONSTRAINT factory_work_order_dispatch_requests_work_order_id_fkey
    FOREIGN KEY (work_order_id) REFERENCES factory_work_orders(id) ON DELETE CASCADE
);

CREATE INDEX idx_factory_work_order_dispatch_requests_organization
  ON factory_work_order_dispatch_requests (organization_id);

CREATE INDEX idx_factory_work_order_dispatch_requests_factory
  ON factory_work_order_dispatch_requests (factory_id);
