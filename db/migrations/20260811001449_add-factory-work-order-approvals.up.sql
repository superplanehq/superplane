BEGIN;

--
-- Approval requests raised against a work order. An approval belongs to a
-- specific work order and, optionally, to a specific execution / phase so the
-- UI can render it inline in a line-run card. The `status` column tracks the
-- pending -> approved/rejected transition; the timeline exposes both the
-- `order.approval.requested` and `order.approval.resolved` events.
--
CREATE TABLE factory_work_order_approvals (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  organization_id UUID NOT NULL,
  factory_id      UUID NOT NULL REFERENCES factories(id) ON DELETE RESTRICT,
  work_order_id   UUID NOT NULL REFERENCES factory_work_orders(id) ON DELETE RESTRICT,
  execution_id    UUID REFERENCES factory_work_order_executions(id) ON DELETE SET NULL,
  title           TEXT NOT NULL DEFAULT '',
  message         TEXT NOT NULL DEFAULT '',
  status          VARCHAR(32) NOT NULL DEFAULT 'pending',
  approver_id     UUID REFERENCES users(id) ON DELETE SET NULL,
  comment         TEXT NOT NULL DEFAULT '',
  resolved_by_id  UUID REFERENCES users(id) ON DELETE SET NULL,
  resolved_at     TIMESTAMPTZ,
  created_by_id   UUID REFERENCES users(id) ON DELETE SET NULL,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_factory_work_order_approvals_work_order_created
  ON factory_work_order_approvals (work_order_id, created_at DESC);

CREATE INDEX idx_factory_work_order_approvals_pending
  ON factory_work_order_approvals (work_order_id)
  WHERE status = 'pending';

COMMIT;
