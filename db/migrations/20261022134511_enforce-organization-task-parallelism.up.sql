CREATE INDEX idx_factory_work_order_executions_organization_active
    ON factory_work_order_executions (organization_id)
    WHERE status IN ('pending', 'running');

CREATE INDEX idx_factory_work_order_queue_items_organization_created
    ON factory_work_order_queue_items (organization_id, created_at, id);
