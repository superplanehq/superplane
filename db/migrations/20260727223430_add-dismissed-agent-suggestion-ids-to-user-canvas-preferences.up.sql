ALTER TABLE public.user_canvas_preferences
  ADD COLUMN dismissed_agent_suggestion_ids jsonb NOT NULL DEFAULT '[]'::jsonb;
