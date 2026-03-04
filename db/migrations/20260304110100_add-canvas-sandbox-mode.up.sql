BEGIN;

ALTER TABLE public.organizations
  ADD COLUMN IF NOT EXISTS canvas_sandbox_mode_enabled boolean DEFAULT true NOT NULL;

COMMIT;
