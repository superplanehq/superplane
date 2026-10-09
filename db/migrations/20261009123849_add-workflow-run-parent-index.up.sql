CREATE INDEX CONCURRENTLY idx_workflow_runs_parent_run_id
ON public.workflow_runs (parent_run_id)
INCLUDE (id)
WHERE parent_run_id IS NOT NULL;
