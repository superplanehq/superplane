BEGIN;

DROP INDEX IF EXISTS idx_user_canvas_preferences_user_pinned;

ALTER TABLE public.user_canvas_preferences
  DROP COLUMN IF EXISTS pinned_at;

COMMIT;
