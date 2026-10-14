CREATE INDEX CONCURRENTLY idx_workflow_events_work_order_camel
ON public.workflow_events ((COALESCE(data->'data', data)->'workOrder'->>'id'), workflow_id)
INCLUDE (run_id)
WHERE COALESCE(data->'data', data)->'workOrder'->>'id' IS NOT NULL;
