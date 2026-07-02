DELETE FROM workflow_staged_files;
DELETE FROM workflow_versions WHERE state = 'draft';

ALTER TABLE workflow_staged_files
    DROP COLUMN base_head_sha;

ALTER TABLE workflow_staged_files
    ADD COLUMN user_id uuid NOT NULL REFERENCES users(id),
    ADD COLUMN workflow_id uuid NOT NULL REFERENCES workflows(id) ON DELETE CASCADE;

ALTER TABLE workflow_staged_files
    DROP CONSTRAINT workflow_staged_files_version_id_path_key;

ALTER TABLE workflow_staged_files
    ADD CONSTRAINT workflow_staged_files_workflow_user_path_key UNIQUE (workflow_id, user_id, path);

CREATE INDEX idx_workflow_staged_files_workflow_user ON workflow_staged_files (workflow_id, user_id);

DROP INDEX IF EXISTS idx_workflow_versions_draft_git_branch;
DROP INDEX IF EXISTS idx_workflow_versions_git_branch;

ALTER TABLE workflow_versions
    DROP COLUMN materialization_status,
    DROP COLUMN materialization_error,
    DROP COLUMN state,
    DROP COLUMN published_at,
    DROP COLUMN git_branch,
    DROP COLUMN display_name;

ALTER TABLE workflow_versions
    ADD COLUMN commit_message text NOT NULL DEFAULT '';

ALTER TABLE workflows
    DROP COLUMN IF EXISTS next_draft_display_number;
