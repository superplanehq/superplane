BEGIN;

ALTER TABLE public.workflow_versions
  ADD COLUMN IF NOT EXISTS state character varying(32) NOT NULL DEFAULT 'draft';

UPDATE public.workflow_versions AS v
SET state = 'published'
WHERE v.id NOT IN (
  SELECT d.version_id FROM public.workflow_user_drafts AS d
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_workflow_versions_unique_draft
  ON public.workflow_versions (workflow_id, owner_id)
  WHERE state = 'draft';

DROP TABLE IF EXISTS public.workflow_user_drafts;

COMMIT;
