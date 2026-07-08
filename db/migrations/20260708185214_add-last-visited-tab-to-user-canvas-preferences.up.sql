BEGIN;

ALTER TABLE public.user_canvas_preferences
  ADD COLUMN IF NOT EXISTS last_visited_tab text;

COMMIT;
