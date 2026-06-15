BEGIN;

-- Git-first canvas versions: commit SHA tracking, materialization state, staging base head.

ALTER TABLE workflow_versions
  ADD COLUMN IF NOT EXISTS commit_sha varchar(40) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS git_branch text NOT NULL DEFAULT 'main',
  ADD COLUMN IF NOT EXISTS materialization_status varchar(32) NOT NULL DEFAULT 'ready',
  ADD COLUMN IF NOT EXISTS materialization_error text NOT NULL DEFAULT '';

ALTER TABLE workflow_versions
  ALTER COLUMN git_branch DROP DEFAULT,
  ALTER COLUMN materialization_status DROP DEFAULT,
  ALTER COLUMN materialization_error DROP DEFAULT;

CREATE INDEX IF NOT EXISTS idx_workflow_versions_commit_sha ON workflow_versions (workflow_id, commit_sha)
  WHERE commit_sha <> '';

CREATE INDEX IF NOT EXISTS idx_workflow_versions_git_branch ON workflow_versions (workflow_id, git_branch);

ALTER TABLE workflow_staged_files
  ADD COLUMN IF NOT EXISTS base_head_sha varchar(40) NOT NULL DEFAULT '';

ALTER TABLE workflow_staged_files
  ALTER COLUMN base_head_sha DROP DEFAULT;

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
