BEGIN;

ALTER TABLE public.organizations
  RENAME COLUMN canvas_sandbox_mode_enabled TO canvas_versioning_enabled;

UPDATE public.organizations
SET canvas_versioning_enabled = NOT canvas_versioning_enabled;

ALTER TABLE public.organizations
  ALTER COLUMN canvas_versioning_enabled SET DEFAULT false;

COMMIT;
