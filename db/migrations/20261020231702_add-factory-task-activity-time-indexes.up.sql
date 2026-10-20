CREATE INDEX idx_factory_work_orders_created_at
ON public.factory_work_orders (created_at);

CREATE INDEX idx_factory_work_order_line_dispatches_created_at
ON public.factory_work_order_line_dispatches (created_at);

CREATE INDEX idx_factory_work_order_events_status_created_at
ON public.factory_work_order_events (created_at)
WHERE type = 'order.status.updated';
