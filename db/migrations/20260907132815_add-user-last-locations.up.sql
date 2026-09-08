BEGIN;

CREATE TABLE IF NOT EXISTS public.user_last_locations (
  organization_id uuid NOT NULL,
  user_id uuid NOT NULL,
  path text NOT NULL,
  created_at timestamp without time zone NOT NULL,
  updated_at timestamp without time zone NOT NULL,
  CONSTRAINT user_last_locations_pkey PRIMARY KEY (organization_id, user_id),
  CONSTRAINT user_last_locations_organization_id_fkey FOREIGN KEY (organization_id) REFERENCES public.organizations(id) ON DELETE CASCADE,
  CONSTRAINT user_last_locations_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE
);

COMMIT;
