CREATE INDEX CONCURRENTLY idx_workflow_runs_work_order_input
ON public.workflow_runs ((input->'work_order'->>'id'), workflow_id)
INCLUDE (id)
WHERE octet_length(input->'work_order'->>'id') <= 36;
