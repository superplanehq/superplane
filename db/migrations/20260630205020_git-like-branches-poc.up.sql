BEGIN;

-- Move canvas description to workflows (name already lives here).
ALTER TABLE workflows
  ADD COLUMN IF NOT EXISTS description text NOT NULL DEFAULT '';

UPDATE workflows w
SET description = COALESCE((
  SELECT v.description
  FROM workflow_versions v
  WHERE v.id = w.live_version_id
), '');

-- Commits carry a message; branches are tracked separately.
ALTER TABLE workflow_versions
  ADD COLUMN IF NOT EXISTS commit_message text NOT NULL DEFAULT '';

UPDATE workflow_versions
SET commit_message = 'Initial commit'
WHERE commit_message = '';

CREATE TABLE IF NOT EXISTS workflow_branches (
  id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
  workflow_id uuid NOT NULL,
  name text NOT NULL,
  head_version_id uuid,
  created_at timestamp without time zone NOT NULL DEFAULT now(),
  updated_at timestamp without time zone NOT NULL DEFAULT now(),
  CONSTRAINT workflow_branches_pkey PRIMARY KEY (id),
  CONSTRAINT workflow_branches_workflow_id_name_key UNIQUE (workflow_id, name),
  CONSTRAINT workflow_branches_workflow_id_fkey FOREIGN KEY (workflow_id) REFERENCES workflows(id) ON DELETE CASCADE,
  CONSTRAINT workflow_branches_head_version_id_fkey FOREIGN KEY (head_version_id) REFERENCES workflow_versions(id) ON DELETE SET NULL
);

CREATE INDEX idx_workflow_branches_workflow_id ON workflow_branches USING btree (workflow_id);

-- Main branch for every canvas.
INSERT INTO workflow_branches (workflow_id, name, head_version_id, created_at, updated_at)
SELECT w.id, 'main', w.live_version_id, w.created_at, w.updated_at
FROM workflows w;

-- Non-main draft branches become named branches (POC: strip drafts/ prefix).
INSERT INTO workflow_branches (workflow_id, name, head_version_id, created_at, updated_at)
SELECT
  v.workflow_id,
  regexp_replace(v.git_branch, '^drafts/', ''),
  v.id,
  v.created_at,
  v.updated_at
FROM workflow_versions v
WHERE v.state = 'draft'
  AND v.git_branch <> ''
  AND v.git_branch <> 'main'
ON CONFLICT (workflow_id, name) DO NOTHING;

-- All published history is on main.
UPDATE workflow_versions
SET git_branch = 'main'
WHERE state = 'published';

UPDATE workflow_versions v
SET commit_message = COALESCE(
  NULLIF(v.commit_message, ''),
  CASE WHEN v.published_at IS NOT NULL THEN 'Published' ELSE 'Commit' END
)
WHERE v.state = 'published';

-- Draft rows become commits on their branch.
UPDATE workflow_versions v
SET commit_message = COALESCE(
  NULLIF(v.commit_message, ''),
  COALESCE(NULLIF(v.display_name, ''), 'WIP commit')
)
WHERE v.state = 'draft';

-- Staging is per branch + user (POC: drop existing staged rows).
DELETE FROM workflow_staged_files;

ALTER TABLE workflow_staged_files
  DROP CONSTRAINT IF EXISTS workflow_staged_files_version_id_path_key,
  DROP CONSTRAINT IF EXISTS workflow_staged_files_version_id_fkey,
  DROP CONSTRAINT IF EXISTS workflow_staged_files_organization_id_fkey;

DROP INDEX IF EXISTS idx_workflow_staged_files_version_id;

ALTER TABLE workflow_staged_files
  DROP COLUMN IF EXISTS version_id,
  DROP COLUMN IF EXISTS organization_id,
  DROP COLUMN IF EXISTS base_head_sha;

ALTER TABLE workflow_staged_files
  ADD COLUMN IF NOT EXISTS branch_id uuid,
  ADD COLUMN IF NOT EXISTS user_id uuid;

ALTER TABLE workflow_staged_files
  ALTER COLUMN branch_id SET NOT NULL,
  ALTER COLUMN user_id SET NOT NULL;

ALTER TABLE workflow_staged_files
  DROP CONSTRAINT IF EXISTS workflow_staged_files_branch_id_fkey,
  DROP CONSTRAINT IF EXISTS workflow_staged_files_user_id_fkey,
  DROP CONSTRAINT IF EXISTS workflow_staged_files_branch_id_user_id_path_key;

ALTER TABLE workflow_staged_files
  ADD CONSTRAINT workflow_staged_files_branch_id_fkey FOREIGN KEY (branch_id) REFERENCES workflow_branches(id) ON DELETE CASCADE,
  ADD CONSTRAINT workflow_staged_files_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  ADD CONSTRAINT workflow_staged_files_branch_id_user_id_path_key UNIQUE (branch_id, user_id, path);

CREATE INDEX IF NOT EXISTS idx_workflow_staged_files_branch_user ON workflow_staged_files USING btree (branch_id, user_id);

DROP INDEX IF EXISTS idx_workflow_versions_draft_git_branch;

ALTER TABLE workflow_versions
  DROP COLUMN IF EXISTS state,
  DROP COLUMN IF EXISTS published_at,
  DROP COLUMN IF EXISTS display_name,
  DROP COLUMN IF EXISTS materialization_status,
  DROP COLUMN IF EXISTS materialization_error,
  DROP COLUMN IF EXISTS name,
  DROP COLUMN IF EXISTS description;

ALTER TABLE workflows
  DROP COLUMN IF EXISTS next_draft_display_number;

COMMIT;
