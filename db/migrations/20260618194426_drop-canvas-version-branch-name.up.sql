BEGIN;

-- Consolidate the redundant branch_name column into git_branch. For draft
-- versions, branch_name and git_branch carry the same value (the drafts/* git
-- branch); the git-first migration defaulted every row's git_branch to 'main',
-- so backfill draft rows from branch_name before dropping it.
UPDATE workflow_versions
  SET git_branch = branch_name
  WHERE branch_name IS NOT NULL AND branch_name <> '';

-- Move the per-draft uniqueness guarantee from branch_name to git_branch and
-- drop the branch_name-based index and check constraint that depend on it.
DROP INDEX IF EXISTS idx_workflow_versions_draft_branch;

ALTER TABLE workflow_versions
  DROP CONSTRAINT IF EXISTS workflow_versions_draft_branch_check;

CREATE UNIQUE INDEX IF NOT EXISTS idx_workflow_versions_draft_git_branch
  ON workflow_versions (workflow_id, git_branch)
  WHERE state = 'draft';

ALTER TABLE workflow_versions
  DROP COLUMN IF EXISTS branch_name;

COMMIT;
