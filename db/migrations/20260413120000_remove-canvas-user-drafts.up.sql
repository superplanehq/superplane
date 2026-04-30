BEGIN;

ALTER TABLE public.workflow_versions
  ADD COLUMN IF NOT EXISTS state character varying(32) NOT NULL DEFAULT 'draft';

UPDATE public.workflow_versions AS v
SET state = 'published'
WHERE v.is_published = true;

UPDATE public.workflow_versions AS v
SET state = 'snapshot'
WHERE v.is_published = false
  AND v.id NOT IN (
    SELECT d.version_id FROM public.workflow_user_drafts AS d
  );

ALTER TABLE public.workflow_versions
  ALTER COLUMN state DROP DEFAULT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_workflow_versions_unique_draft
  ON public.workflow_versions (workflow_id, owner_id)
  WHERE state = 'draft';

DROP TABLE IF EXISTS public.workflow_user_drafts;

-- Drop the now-redundant is_published column and its index.
DROP INDEX IF EXISTS idx_workflow_versions_published;

ALTER TABLE public.workflow_versions
  DROP COLUMN IF EXISTS is_published;

COMMIT;
