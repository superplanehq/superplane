BEGIN;

ALTER TABLE public.workflows
  DROP CONSTRAINT IF EXISTS workflows_folder_id_fkey;

DROP INDEX IF EXISTS idx_workflows_folder_id;

ALTER TABLE public.workflows
  DROP COLUMN IF EXISTS folder_id;

DROP INDEX IF EXISTS idx_canvas_folders_organization_id_title;

DROP TABLE IF EXISTS public.canvas_folders;

COMMIT;
