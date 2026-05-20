BEGIN;

ALTER TABLE public.workflow_runs
  ADD COLUMN version_id uuid;

UPDATE public.workflow_runs AS r
SET version_id = COALESCE(
  (
    SELECT v.id
    FROM public.workflow_versions AS v
    WHERE v.workflow_id = r.workflow_id
      AND v.state = 'published'
      AND v.published_at <= r.created_at
    ORDER BY v.published_at DESC, v.created_at DESC
    LIMIT 1
  ),
  w.live_version_id
)
FROM public.workflows AS w
WHERE w.id = r.workflow_id;

ALTER TABLE public.workflow_runs
  ALTER COLUMN version_id SET NOT NULL;

ALTER TABLE public.workflow_runs
  ADD CONSTRAINT workflow_runs_version_id_fkey
  FOREIGN KEY (version_id) REFERENCES public.workflow_versions(id) ON DELETE RESTRICT;

CREATE INDEX idx_workflow_runs_version_id ON public.workflow_runs(version_id);

COMMIT;
