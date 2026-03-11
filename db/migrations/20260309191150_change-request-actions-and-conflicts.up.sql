BEGIN;

ALTER TABLE public.workflow_change_requests
  ADD COLUMN IF NOT EXISTS based_on_version_id uuid,
  ADD COLUMN IF NOT EXISTS conflicting_node_ids jsonb DEFAULT '[]'::jsonb NOT NULL;

ALTER TABLE ONLY public.workflow_change_requests
  DROP CONSTRAINT IF EXISTS workflow_change_requests_based_on_version_id_fkey;

ALTER TABLE ONLY public.workflow_change_requests
  ADD CONSTRAINT workflow_change_requests_based_on_version_id_fkey
  FOREIGN KEY (based_on_version_id) REFERENCES public.workflow_versions(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_workflow_change_requests_based_on_version
  ON public.workflow_change_requests (workflow_id, based_on_version_id);

COMMIT;
