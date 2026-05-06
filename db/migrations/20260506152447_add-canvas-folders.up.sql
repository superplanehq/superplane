BEGIN;

CREATE TABLE IF NOT EXISTS public.canvas_folders (
  id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
  organization_id uuid NOT NULL,
  title character varying(128) NOT NULL,
  background_color character varying(32) NOT NULL DEFAULT 'color_1',
  sort_order bigint NOT NULL,
  created_at timestamp without time zone NOT NULL,
  updated_at timestamp without time zone NOT NULL,
  CONSTRAINT canvas_folders_pkey PRIMARY KEY (id),
  CONSTRAINT canvas_folders_organization_id_title_key UNIQUE (organization_id, title),
  CONSTRAINT canvas_folders_organization_id_fkey FOREIGN KEY (organization_id) REFERENCES public.organizations(id) ON DELETE CASCADE,
  CONSTRAINT canvas_folders_background_color_check CHECK (background_color IN ('color_1', 'color_2', 'color_3', 'color_4', 'color_5', 'color_6'))
);

CREATE INDEX IF NOT EXISTS idx_canvas_folders_organization_id_title
  ON public.canvas_folders (organization_id, title);

ALTER TABLE public.workflows
  ADD COLUMN IF NOT EXISTS canvas_folder_id uuid;

ALTER TABLE public.workflows
  ADD CONSTRAINT workflows_canvas_folder_id_fkey
  FOREIGN KEY (canvas_folder_id) REFERENCES public.canvas_folders(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_workflows_canvas_folder_id
  ON public.workflows (canvas_folder_id);

COMMIT;
