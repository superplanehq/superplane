BEGIN;

-- Introduce the multi-page console shape as a single JSONB column and
-- retire the flat `console_panels` / `console_layout` pair. Existing
-- consoles are backfilled into one implicit `main` page so on-disk data
-- stays fully preserved (over-cap consoles are grandfathered at read
-- time; validation only fires on new imports/commits).
ALTER TABLE public.workflow_versions
  ADD COLUMN IF NOT EXISTS console_pages JSONB NOT NULL DEFAULT '[]'::jsonb;

UPDATE public.workflow_versions
SET console_pages = jsonb_build_array(
  jsonb_build_object(
    'id',     'main',
    'name',   'Main',
    'panels', COALESCE(console_panels, '[]'::jsonb),
    'layout', COALESCE(console_layout, '[]'::jsonb)
  )
)
WHERE (
    COALESCE(jsonb_array_length(console_panels), 0) > 0
 OR COALESCE(jsonb_array_length(console_layout), 0) > 0
);

ALTER TABLE public.workflow_versions
  DROP COLUMN IF EXISTS console_panels,
  DROP COLUMN IF EXISTS console_layout;

COMMIT;
