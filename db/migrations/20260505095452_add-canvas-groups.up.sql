BEGIN;

CREATE TABLE IF NOT EXISTS public.canvas_groups (
  id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
  organization_id uuid NOT NULL,
  title character varying(128) NOT NULL,
  background_color character varying(32) NOT NULL DEFAULT 'blue-800',
  created_at timestamp without time zone NOT NULL,
  updated_at timestamp without time zone NOT NULL,
  CONSTRAINT canvas_groups_pkey PRIMARY KEY (id),
  CONSTRAINT canvas_groups_organization_id_title_key UNIQUE (organization_id, title),
  CONSTRAINT canvas_groups_organization_id_fkey FOREIGN KEY (organization_id) REFERENCES public.organizations(id) ON DELETE CASCADE,
  CONSTRAINT canvas_groups_background_color_check CHECK (background_color IN ('blue-800', 'green-800', 'violet-800', 'yellow-800'))
);

CREATE INDEX IF NOT EXISTS idx_canvas_groups_organization_id_title
  ON public.canvas_groups (organization_id, title);

ALTER TABLE public.workflows
  ADD COLUMN IF NOT EXISTS canvas_group_id uuid;

ALTER TABLE public.workflows
  ADD CONSTRAINT workflows_canvas_group_id_fkey
  FOREIGN KEY (canvas_group_id) REFERENCES public.canvas_groups(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_workflows_canvas_group_id
  ON public.workflows (canvas_group_id);

COMMIT;
