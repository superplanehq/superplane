BEGIN;

-- Add state column to workflow_versions.
-- Existing rows default to 'published'.
ALTER TABLE public.workflow_versions
  ADD COLUMN IF NOT EXISTS state character varying(32) NOT NULL DEFAULT 'published';

-- Migrate: mark versions referenced in workflow_user_drafts as 'draft'.
UPDATE public.workflow_versions AS v
SET state = 'draft'
FROM public.workflow_user_drafts AS d
WHERE v.id = d.version_id;

-- One draft per user per canvas.
CREATE UNIQUE INDEX IF NOT EXISTS idx_workflow_versions_unique_draft
  ON public.workflow_versions (workflow_id, owner_id)
  WHERE state = 'draft';

-- Drop the now-redundant drafts table (cascades FK constraints and indexes).
DROP TABLE IF EXISTS public.workflow_user_drafts;

COMMIT;
