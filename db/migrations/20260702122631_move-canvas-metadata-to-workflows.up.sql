BEGIN;

ALTER TABLE public.workflows
  ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT '';

UPDATE public.workflows AS w
SET
  name = COALESCE(NULLIF(lv.name, ''), w.name, ''),
  description = COALESCE(lv.description, '')
FROM public.workflow_versions AS lv
WHERE lv.id = w.live_version_id;

ALTER TABLE public.workflow_versions
  DROP COLUMN IF EXISTS name,
  DROP COLUMN IF EXISTS description;

COMMIT;
