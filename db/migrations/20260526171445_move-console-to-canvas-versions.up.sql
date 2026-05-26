BEGIN;

ALTER TABLE public.workflow_versions
  ADD COLUMN IF NOT EXISTS console_panels JSONB NOT NULL DEFAULT '[]'::jsonb,
  ADD COLUMN IF NOT EXISTS console_layout JSONB NOT NULL DEFAULT '[]'::jsonb;

DROP TABLE IF EXISTS public.canvas_dashboards;

COMMIT;
