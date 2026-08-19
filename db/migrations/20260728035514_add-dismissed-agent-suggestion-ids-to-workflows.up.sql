ALTER TABLE public.workflows
  ADD COLUMN dismissed_agent_suggestion_ids jsonb NOT NULL DEFAULT '[]'::jsonb;
