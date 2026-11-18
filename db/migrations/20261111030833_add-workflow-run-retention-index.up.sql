CREATE INDEX CONCURRENTLY idx_workflow_runs_retention_finished_at
ON public.workflow_runs (finished_at)
WHERE state = 'finished' AND finished_at IS NOT NULL;
