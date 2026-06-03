BEGIN;

-- Git-first canvas versions: commit SHA as version id, draft branches, materialization state.

DROP INDEX IF EXISTS idx_workflow_versions_unique_draft;

ALTER TABLE workflow_change_requests
  DROP CONSTRAINT IF EXISTS workflow_change_requests_version_id_fkey,
  DROP CONSTRAINT IF EXISTS workflow_change_requests_based_on_version_id_fkey;

ALTER TABLE workflow_runs
  DROP CONSTRAINT IF EXISTS workflow_runs_version_id_fkey;

ALTER TABLE workflows
  DROP CONSTRAINT IF EXISTS workflows_live_version_id_fkey;

ALTER TABLE workflow_versions
  DROP CONSTRAINT IF EXISTS workflow_versions_pkey;

ALTER TABLE workflow_versions
  ALTER COLUMN id DROP DEFAULT;

ALTER TABLE workflow_versions
  ALTER COLUMN id TYPE varchar(40) USING id::text;

ALTER TABLE workflow_versions
  ADD COLUMN IF NOT EXISTS git_branch text NOT NULL DEFAULT 'main',
  ADD COLUMN IF NOT EXISTS materialization_status varchar(32) NOT NULL DEFAULT 'ready',
  ADD COLUMN IF NOT EXISTS materialization_error text NOT NULL DEFAULT '';

ALTER TABLE workflow_versions
  ALTER COLUMN git_branch DROP DEFAULT;

ALTER TABLE workflows
  ALTER COLUMN live_version_id TYPE varchar(40) USING live_version_id::text;

ALTER TABLE workflow_runs
  ALTER COLUMN version_id TYPE varchar(40) USING version_id::text;

ALTER TABLE workflow_change_requests
  ALTER COLUMN version_id TYPE varchar(40) USING version_id::text,
  ALTER COLUMN based_on_version_id TYPE varchar(40) USING based_on_version_id::text;

ALTER TABLE workflow_change_requests
  ADD COLUMN IF NOT EXISTS draft_branch text NOT NULL DEFAULT '';

ALTER TABLE workflow_change_requests
  ALTER COLUMN draft_branch DROP DEFAULT;

ALTER TABLE workflow_versions
  ADD CONSTRAINT workflow_versions_pkey PRIMARY KEY (id);

ALTER TABLE workflow_change_requests
  ADD CONSTRAINT workflow_change_requests_version_id_fkey
    FOREIGN KEY (version_id) REFERENCES workflow_versions(id) ON DELETE CASCADE;

ALTER TABLE workflow_change_requests
  ADD CONSTRAINT workflow_change_requests_based_on_version_id_fkey
    FOREIGN KEY (based_on_version_id) REFERENCES workflow_versions(id) ON DELETE SET NULL;

ALTER TABLE workflow_runs
  ADD CONSTRAINT workflow_runs_version_id_fkey
    FOREIGN KEY (version_id) REFERENCES workflow_versions(id) ON DELETE RESTRICT;

ALTER TABLE workflows
  ADD CONSTRAINT workflows_live_version_id_fkey
    FOREIGN KEY (live_version_id) REFERENCES workflow_versions(id) ON DELETE RESTRICT DEFERRABLE INITIALLY DEFERRED;

CREATE INDEX IF NOT EXISTS idx_workflow_versions_git_branch ON workflow_versions (workflow_id, git_branch);

CREATE TABLE IF NOT EXISTS canvas_draft_branches (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  canvas_id       UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  branch_name     TEXT NOT NULL,
  display_name    TEXT NOT NULL DEFAULT '',
  owner_id        UUID REFERENCES users(id) ON DELETE SET NULL,
  created_by      UUID REFERENCES users(id) ON DELETE SET NULL,
  tip_sha         varchar(40) NOT NULL DEFAULT '',
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  UNIQUE (canvas_id, branch_name)
);

CREATE INDEX IF NOT EXISTS idx_canvas_draft_branches_canvas_id ON canvas_draft_branches (canvas_id);
CREATE INDEX IF NOT EXISTS idx_canvas_draft_branches_owner_id ON canvas_draft_branches (owner_id);

CREATE TABLE IF NOT EXISTS repository_materialization_state (
  id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  canvas_id        UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
  branch           TEXT NOT NULL,
  head_sha         varchar(40) NOT NULL DEFAULT '',
  materialized_sha varchar(40) NOT NULL DEFAULT '',
  status           varchar(32) NOT NULL DEFAULT 'pending',
  error            TEXT NOT NULL DEFAULT '',
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  UNIQUE (canvas_id, branch)
);

CREATE INDEX IF NOT EXISTS idx_repository_materialization_state_canvas_id ON repository_materialization_state (canvas_id);

COMMIT;
