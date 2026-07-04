BEGIN;

--
-- Remove currently staged files and state=draft versions
-- workflow_versions records are "commits on main branch" now
--
DELETE FROM workflow_staged_files;
DELETE FROM workflow_versions WHERE state = 'draft';

--
-- Remove unused column, and rename version_id to base_version_id,
-- to reflect its new purpose: the "base version" from where this staged file was created.
--
ALTER TABLE workflow_staged_files
    DROP CONSTRAINT workflow_staged_files_version_id_path_key;

DROP INDEX IF EXISTS idx_workflow_staged_files_version_id;

ALTER TABLE workflow_staged_files
    DROP COLUMN base_head_sha;

ALTER TABLE workflow_staged_files
    RENAME COLUMN version_id TO base_version_id;

--
-- Staging is now scoped to a workflow and a user
--
ALTER TABLE workflow_staged_files
    DROP COLUMN updated_by,
    ADD COLUMN user_id uuid NOT NULL REFERENCES users(id),
    ADD COLUMN workflow_id uuid NOT NULL REFERENCES workflows(id) ON DELETE CASCADE;

--
-- TODO: verify if this constraint update makes sense
--
ALTER TABLE workflow_staged_files
    ADD CONSTRAINT workflow_staged_files_workflow_user_path_key UNIQUE (workflow_id, user_id, path);

--
-- Now that staged files are per user, we need an index on workflow_staged_files for faster lookups
--
CREATE INDEX idx_workflow_staged_files_workflow_user ON workflow_staged_files (workflow_id, user_id);

--
-- Drop indexes for workflow_versions columns we don't need anymore
--
DROP INDEX IF EXISTS idx_workflow_versions_draft_git_branch;
DROP INDEX IF EXISTS idx_workflow_versions_git_branch;

--
-- Remove unused columns from workflow_versions
--
ALTER TABLE workflow_versions
    DROP COLUMN materialization_status,
    DROP COLUMN materialization_error,
    DROP COLUMN state,
    DROP COLUMN published_at,
    DROP COLUMN git_branch,
    DROP COLUMN display_name;

--
-- workflow_versions records are "commits" now, so they need a commit message
--
ALTER TABLE workflow_versions
    ADD COLUMN commit_message text NOT NULL DEFAULT '';

--
-- Now that we don't have multiple drafts,
-- we don't need the next_draft_display_number column for generating display names anymore.
--
ALTER TABLE workflows
    DROP COLUMN IF EXISTS next_draft_display_number;

COMMIT;
