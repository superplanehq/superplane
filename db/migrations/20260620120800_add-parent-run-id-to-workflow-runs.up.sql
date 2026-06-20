BEGIN;

ALTER TABLE public.workflow_runs
  ADD COLUMN parent_run_id uuid,
  ADD COLUMN spawned_by_execution_id uuid;

ALTER TABLE public.workflow_runs
  ADD CONSTRAINT workflow_runs_parent_run_id_fkey
  FOREIGN KEY (parent_run_id) REFERENCES public.workflow_runs(id) ON DELETE SET NULL;

ALTER TABLE public.workflow_runs
  ADD CONSTRAINT workflow_runs_spawned_by_execution_id_fkey
  FOREIGN KEY (spawned_by_execution_id) REFERENCES public.workflow_node_executions(id) ON DELETE SET NULL;

CREATE INDEX idx_workflow_runs_parent_run_id ON public.workflow_runs(parent_run_id);
CREATE INDEX idx_workflow_runs_spawned_by_execution_id ON public.workflow_runs(spawned_by_execution_id);

COMMIT;
