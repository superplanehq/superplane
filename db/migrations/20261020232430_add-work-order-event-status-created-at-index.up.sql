CREATE INDEX CONCURRENTLY idx_factory_work_order_events_status_created_at
ON public.factory_work_order_events (created_at)
WHERE type = 'order.status.updated';
