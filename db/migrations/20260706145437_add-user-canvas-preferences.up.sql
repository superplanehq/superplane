BEGIN;

CREATE TABLE IF NOT EXISTS public.user_canvas_preferences (
  organization_id uuid NOT NULL,
  user_id uuid NOT NULL,
  canvas_id uuid NOT NULL,
  pinned_at timestamp without time zone,
  starred_at timestamp without time zone,
  created_at timestamp without time zone NOT NULL,
  updated_at timestamp without time zone NOT NULL,
  CONSTRAINT user_canvas_preferences_pkey PRIMARY KEY (organization_id, user_id, canvas_id),
  CONSTRAINT user_canvas_preferences_organization_id_fkey FOREIGN KEY (organization_id) REFERENCES public.organizations(id) ON DELETE CASCADE,
  CONSTRAINT user_canvas_preferences_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE,
  CONSTRAINT user_canvas_preferences_canvas_id_fkey FOREIGN KEY (canvas_id) REFERENCES public.workflows(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_user_canvas_preferences_user_pinned
  ON public.user_canvas_preferences (organization_id, user_id, pinned_at DESC)
  WHERE pinned_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_user_canvas_preferences_user_starred
  ON public.user_canvas_preferences (organization_id, user_id, starred_at DESC)
  WHERE starred_at IS NOT NULL;

COMMIT;
