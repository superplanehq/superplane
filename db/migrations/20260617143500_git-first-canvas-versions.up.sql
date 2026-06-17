BEGIN;

-- Git-first canvas versions: commit SHA tracking, materialization status, staging base head.

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

COMMIT;
