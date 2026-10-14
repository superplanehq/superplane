CREATE INDEX CONCURRENTLY idx_workflow_events_work_order_camel
ON public.workflow_events ((COALESCE(data->'data', data)->'workOrder'->>'id'), workflow_id)
INCLUDE (run_id)
WHERE octet_length(COALESCE(data->'data', data)->'workOrder'->>'id') <= 36;
