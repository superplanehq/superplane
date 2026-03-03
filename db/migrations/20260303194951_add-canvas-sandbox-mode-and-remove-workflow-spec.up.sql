ALTER TABLE public.organizations
    ADD COLUMN canvas_sandbox_mode_enabled boolean DEFAULT true NOT NULL;

WITH latest_versions AS (
    SELECT DISTINCT ON (workflow_id)
        workflow_id,
        id
    FROM public.workflow_versions
    ORDER BY workflow_id, is_published DESC, revision DESC, created_at DESC, id DESC
)
UPDATE public.workflows AS w
SET live_version_id = lv.id
FROM latest_versions AS lv
WHERE w.id = lv.workflow_id
  AND w.live_version_id IS NULL;

WITH targets AS (
    SELECT
        w.id AS workflow_id,
        COALESCE(MAX(v.revision), 0) + 1 AS revision,
        w.created_by AS owner_id,
        w.nodes,
        w.edges
    FROM public.workflows AS w
    LEFT JOIN public.workflow_versions AS v
        ON v.workflow_id = w.id
    WHERE w.live_version_id IS NULL
    GROUP BY w.id, w.created_by, w.nodes, w.edges
),
inserted AS (
    INSERT INTO public.workflow_versions (
        id,
        workflow_id,
        revision,
        owner_id,
        based_on_version_id,
        is_published,
        published_at,
        nodes,
        edges,
        created_at,
        updated_at
    )
    SELECT
        public.uuid_generate_v4(),
        t.workflow_id,
        t.revision,
        t.owner_id,
        NULL,
        true,
        NOW(),
        t.nodes,
        t.edges,
        NOW(),
        NOW()
    FROM targets AS t
    RETURNING workflow_id, id
)
UPDATE public.workflows AS w
SET live_version_id = i.id
FROM inserted AS i
WHERE w.id = i.workflow_id;

ALTER TABLE ONLY public.workflows
    ALTER COLUMN live_version_id SET NOT NULL;

ALTER TABLE ONLY public.workflows
    DROP CONSTRAINT IF EXISTS workflows_live_version_id_fkey;

ALTER TABLE ONLY public.workflows
    ADD CONSTRAINT workflows_live_version_id_fkey
    FOREIGN KEY (live_version_id)
    REFERENCES public.workflow_versions(id)
    ON DELETE RESTRICT;

ALTER TABLE ONLY public.workflows
    DROP COLUMN IF EXISTS nodes,
    DROP COLUMN IF EXISTS edges;
