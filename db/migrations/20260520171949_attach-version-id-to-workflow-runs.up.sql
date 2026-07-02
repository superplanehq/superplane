BEGIN;

ALTER TABLE public.workflow_runs
  ADD COLUMN version_id uuid;

UPDATE public.workflow_runs AS r
SET version_id = (
  SELECT v.id
  FROM public.workflow_versions AS v
  WHERE v.workflow_id = r.workflow_id
  ORDER BY
    CASE
      WHEN v.state = 'published' AND v.published_at <= r.created_at THEN 0
      WHEN v.state = 'published' THEN 1
      ELSE 2
    END,
    CASE
      WHEN v.state = 'published' AND v.published_at <= r.created_at THEN v.published_at
    END DESC NULLS LAST,
    CASE
      WHEN v.state = 'published' THEN v.published_at
    END ASC NULLS LAST,
    v.created_at ASC,
    v.id ASC
  LIMIT 1
);

ALTER TABLE public.workflow_runs
  ALTER COLUMN version_id SET NOT NULL;

ALTER TABLE public.workflow_runs
  ADD CONSTRAINT workflow_runs_version_id_fkey
  FOREIGN KEY (version_id) REFERENCES public.workflow_versions(id) ON DELETE RESTRICT;

CREATE INDEX idx_workflow_runs_version_id ON public.workflow_runs(version_id);

COMMIT;
