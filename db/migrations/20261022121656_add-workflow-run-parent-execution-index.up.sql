CREATE INDEX IF NOT EXISTS idx_workflow_runs_parent_execution
ON public.workflow_runs (parent_workflow_id, parent_execution_id)
WHERE parent_execution_id IS NOT NULL;
