-- This branch's task-admission migration did not ship. These indexes remain
-- useful for per-step queues; use a forward version after the current main.
CREATE INDEX IF NOT EXISTS idx_factory_work_order_executions_organization_active
    ON factory_work_order_executions (organization_id)
    WHERE status IN ('pending', 'running');

CREATE INDEX IF NOT EXISTS idx_factory_work_order_queue_items_organization_created
    ON factory_work_order_queue_items (organization_id, created_at, id);
