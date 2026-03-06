BEGIN;

ALTER TABLE public.workflows
  ADD COLUMN IF NOT EXISTS live_version_id uuid;

CREATE TABLE IF NOT EXISTS public.workflow_versions (
  id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
  workflow_id uuid NOT NULL,
  owner_id uuid,
  is_published boolean DEFAULT false NOT NULL,
  published_at timestamp without time zone,
  nodes jsonb DEFAULT '[]'::jsonb NOT NULL,
  edges jsonb DEFAULT '[]'::jsonb NOT NULL,
  created_at timestamp without time zone NOT NULL,
  updated_at timestamp without time zone NOT NULL,
  CONSTRAINT workflow_versions_pkey PRIMARY KEY (id),
  CONSTRAINT workflow_versions_workflow_id_fkey FOREIGN KEY (workflow_id) REFERENCES public.workflows(id) ON DELETE CASCADE,
  CONSTRAINT workflow_versions_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES public.users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_workflow_versions_workflow_id ON public.workflow_versions (workflow_id);
CREATE INDEX IF NOT EXISTS idx_workflow_versions_published ON public.workflow_versions (workflow_id, is_published, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_workflow_versions_owner ON public.workflow_versions (owner_id);

CREATE TABLE IF NOT EXISTS public.workflow_user_drafts (
  workflow_id uuid NOT NULL,
  user_id uuid NOT NULL,
  version_id uuid NOT NULL,
  created_at timestamp without time zone NOT NULL,
  updated_at timestamp without time zone NOT NULL,
  CONSTRAINT workflow_user_drafts_pkey PRIMARY KEY (workflow_id, user_id),
  CONSTRAINT workflow_user_drafts_workflow_id_fkey FOREIGN KEY (workflow_id) REFERENCES public.workflows(id) ON DELETE CASCADE,
  CONSTRAINT workflow_user_drafts_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE,
  CONSTRAINT workflow_user_drafts_version_id_fkey FOREIGN KEY (version_id) REFERENCES public.workflow_versions(id) ON DELETE CASCADE,
  CONSTRAINT workflow_user_drafts_version_id_key UNIQUE (version_id)
);

CREATE INDEX IF NOT EXISTS idx_workflow_user_drafts_user_id ON public.workflow_user_drafts (user_id);

INSERT INTO public.workflow_versions (
  id,
  workflow_id,
  owner_id,
  is_published,
  published_at,
  nodes,
  edges,
  created_at,
  updated_at
)
SELECT
  public.uuid_generate_v4(),
  w.id,
  w.created_by,
  true,
  COALESCE(w.updated_at, NOW()),
  w.nodes,
  w.edges,
  COALESCE(w.created_at, NOW()),
  COALESCE(w.updated_at, NOW())
FROM public.workflows AS w
WHERE NOT EXISTS (
  SELECT 1
  FROM public.workflow_versions AS v
  WHERE v.workflow_id = w.id
);

UPDATE public.workflows AS w
SET live_version_id = v.id
FROM (
  SELECT DISTINCT ON (workflow_id)
    workflow_id,
    id
  FROM public.workflow_versions
  ORDER BY workflow_id, is_published DESC, published_at DESC NULLS LAST, created_at DESC
) AS v
WHERE v.workflow_id = w.id
  AND w.live_version_id IS NULL;

ALTER TABLE ONLY public.workflows
  ALTER COLUMN live_version_id SET NOT NULL;

ALTER TABLE ONLY public.workflows
  DROP CONSTRAINT IF EXISTS workflows_live_version_id_fkey;

ALTER TABLE ONLY public.workflows
  ADD CONSTRAINT workflows_live_version_id_fkey
  FOREIGN KEY (live_version_id)
  REFERENCES public.workflow_versions(id)
  ON DELETE RESTRICT
  DEFERRABLE INITIALLY DEFERRED;

CREATE INDEX IF NOT EXISTS idx_workflows_live_version_id ON public.workflows (live_version_id);

CREATE TABLE IF NOT EXISTS public.workflow_change_requests (
  id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
  workflow_id uuid NOT NULL,
  version_id uuid NOT NULL,
  owner_id uuid,
  status character varying(32) NOT NULL,
  changed_node_ids jsonb DEFAULT '[]'::jsonb NOT NULL,
  title text NOT NULL DEFAULT '',
  description text NOT NULL DEFAULT '',
  published_at timestamp without time zone,
  created_at timestamp without time zone NOT NULL,
  updated_at timestamp without time zone NOT NULL,
  CONSTRAINT workflow_change_requests_pkey PRIMARY KEY (id),
  CONSTRAINT workflow_change_requests_workflow_version_key UNIQUE (workflow_id, version_id),
  CONSTRAINT workflow_change_requests_workflow_id_fkey FOREIGN KEY (workflow_id) REFERENCES public.workflows(id) ON DELETE CASCADE,
  CONSTRAINT workflow_change_requests_version_id_fkey FOREIGN KEY (version_id) REFERENCES public.workflow_versions(id) ON DELETE CASCADE,
  CONSTRAINT workflow_change_requests_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES public.users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_workflow_change_requests_workflow_id ON public.workflow_change_requests (workflow_id);
CREATE INDEX IF NOT EXISTS idx_workflow_change_requests_status ON public.workflow_change_requests (workflow_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_workflow_change_requests_owner ON public.workflow_change_requests (owner_id);

ALTER TABLE ONLY public.workflows
  DROP COLUMN IF EXISTS nodes,
  DROP COLUMN IF EXISTS edges;

COMMIT;
