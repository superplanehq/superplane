ALTER TABLE workflows
ADD COLUMN live_version_id uuid;

CREATE TABLE workflow_versions (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    workflow_id uuid NOT NULL,
    revision integer NOT NULL,
    owner_id uuid,
    based_on_version_id uuid,
    is_published boolean DEFAULT false NOT NULL,
    published_at timestamp without time zone,
    nodes jsonb DEFAULT '[]'::jsonb NOT NULL,
    edges jsonb DEFAULT '[]'::jsonb NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    CONSTRAINT workflow_versions_pkey PRIMARY KEY (id),
    CONSTRAINT workflow_versions_workflow_revision_key UNIQUE (workflow_id, revision),
    CONSTRAINT workflow_versions_workflow_id_fkey FOREIGN KEY (workflow_id) REFERENCES workflows(id) ON DELETE CASCADE,
    CONSTRAINT workflow_versions_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT workflow_versions_based_on_version_id_fkey FOREIGN KEY (based_on_version_id) REFERENCES workflow_versions(id) ON DELETE SET NULL
);

CREATE INDEX idx_workflow_versions_workflow_id ON workflow_versions (workflow_id);
CREATE INDEX idx_workflow_versions_published ON workflow_versions (workflow_id, is_published, created_at DESC);
CREATE INDEX idx_workflow_versions_owner ON workflow_versions (owner_id);

CREATE TABLE workflow_user_drafts (
    workflow_id uuid NOT NULL,
    user_id uuid NOT NULL,
    version_id uuid NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    CONSTRAINT workflow_user_drafts_pkey PRIMARY KEY (workflow_id, user_id),
    CONSTRAINT workflow_user_drafts_workflow_id_fkey FOREIGN KEY (workflow_id) REFERENCES workflows(id) ON DELETE CASCADE,
    CONSTRAINT workflow_user_drafts_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT workflow_user_drafts_version_id_fkey FOREIGN KEY (version_id) REFERENCES workflow_versions(id) ON DELETE CASCADE,
    CONSTRAINT workflow_user_drafts_version_id_key UNIQUE (version_id)
);

CREATE INDEX idx_workflow_user_drafts_user_id ON workflow_user_drafts (user_id);

INSERT INTO workflow_versions (
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
    uuid_generate_v4(),
    w.id,
    1,
    w.created_by,
    NULL,
    true,
    COALESCE(w.updated_at, now()),
    w.nodes,
    w.edges,
    COALESCE(w.created_at, now()),
    COALESCE(w.updated_at, now())
FROM workflows w;

UPDATE workflows w
SET live_version_id = v.id
FROM workflow_versions v
WHERE v.workflow_id = w.id
  AND v.revision = 1
  AND w.live_version_id IS NULL;

ALTER TABLE workflows
ADD CONSTRAINT workflows_live_version_id_fkey
FOREIGN KEY (live_version_id) REFERENCES workflow_versions(id) ON DELETE SET NULL;

CREATE INDEX idx_workflows_live_version_id ON workflows (live_version_id);
