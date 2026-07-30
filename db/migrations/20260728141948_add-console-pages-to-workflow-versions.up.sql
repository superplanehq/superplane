BEGIN;

-- Introduce the multi-page console shape as a single JSONB column and
-- retire the flat `console_panels` / `console_layout` pair. Existing
-- consoles are backfilled into one implicit `main` page so on-disk data
-- stays fully preserved (over-cap consoles are grandfathered at read
-- time; validation only fires on new imports/commits).
ALTER TABLE public.workflow_versions
  ADD COLUMN IF NOT EXISTS console_pages JSONB NOT NULL DEFAULT '[]'::jsonb;

-- `console_panels` / `console_layout` are declared NOT NULL DEFAULT '[]'::jsonb,
-- but historical rows in existing databases can still hold non-array values
-- (objects, scalars, or nulls before the NOT NULL default was enforced).
-- `jsonb_array_length` panics on anything that isn't a JSON array, so we
-- gate every access with `jsonb_typeof = 'array'` and fall back to '[]'::jsonb.
-- The row is only wrapped into a `main` page when there is real content —
-- empty rows keep the default `[]::jsonb` and are treated as no-console.
UPDATE public.workflow_versions
SET console_pages = jsonb_build_array(
  jsonb_build_object(
    'id',     'main',
    'name',   'Main',
    'panels', CASE WHEN jsonb_typeof(console_panels) = 'array' THEN console_panels ELSE '[]'::jsonb END,
    'layout', CASE WHEN jsonb_typeof(console_layout) = 'array' THEN console_layout ELSE '[]'::jsonb END
  )
)
WHERE (
    (jsonb_typeof(console_panels) = 'array' AND jsonb_array_length(console_panels) > 0)
 OR (jsonb_typeof(console_layout) = 'array' AND jsonb_array_length(console_layout) > 0)
);

ALTER TABLE public.workflow_versions
  DROP COLUMN IF EXISTS console_panels,
  DROP COLUMN IF EXISTS console_layout;

COMMIT;
