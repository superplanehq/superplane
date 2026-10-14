CREATE INDEX CONCURRENTLY idx_workflow_events_work_order
ON public.workflow_events ((COALESCE(data->'data', data)->'work_order'->>'id'), workflow_id)
INCLUDE (run_id)
WHERE COALESCE(data->'data', data)->'work_order'->>'id' IS NOT NULL;
