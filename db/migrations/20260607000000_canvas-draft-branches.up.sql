BEGIN;

DROP INDEX IF EXISTS idx_workflow_versions_unique_draft;

ALTER TABLE workflows
  ADD COLUMN next_draft_display_number integer NOT NULL DEFAULT 1;

ALTER TABLE workflow_versions
  ADD COLUMN branch_name TEXT,
  ADD COLUMN display_name TEXT NOT NULL DEFAULT '';

UPDATE workflow_versions AS wv
SET
  branch_name = 'drafts/' || gen_random_uuid()::text,
  display_name = 'Draft #' || numbered.row_num::text
FROM (
  SELECT
    id,
    ROW_NUMBER() OVER (
      PARTITION BY workflow_id
      ORDER BY created_at ASC, id ASC
    ) AS row_num
  FROM workflow_versions
  WHERE state = 'draft'
) AS numbered
WHERE wv.id = numbered.id;

UPDATE workflows AS w
SET next_draft_display_number = sub.next_num
FROM (
  SELECT workflow_id, COUNT(*)::integer + 1 AS next_num
  FROM workflow_versions
  WHERE state = 'draft' AND branch_name IS NOT NULL
  GROUP BY workflow_id
) AS sub
WHERE w.id = sub.workflow_id;

CREATE UNIQUE INDEX IF NOT EXISTS idx_workflow_versions_draft_branch
  ON workflow_versions (workflow_id, branch_name)
  WHERE state = 'draft' AND branch_name IS NOT NULL;

ALTER TABLE workflow_versions
  ADD CONSTRAINT workflow_versions_draft_branch_check
  CHECK (
    (state = 'draft' AND branch_name IS NOT NULL)
    OR (state <> 'draft' AND branch_name IS NULL)
  );

COMMIT;
