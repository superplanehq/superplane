CREATE INDEX CONCURRENTLY idx_workflow_runs_work_order_input
ON public.workflow_runs ((input->'work_order'->>'id'), workflow_id)
INCLUDE (id)
WHERE input->'work_order'->>'id' IS NOT NULL;
