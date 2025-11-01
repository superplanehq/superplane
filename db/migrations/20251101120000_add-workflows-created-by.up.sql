begin;

ALTER TABLE workflows
  ADD COLUMN IF NOT EXISTS created_by uuid;

-- Backfill: set created_by to the first user (by created_at)
-- in the same organization as the workflow.
UPDATE workflows w
SET created_by = (
  SELECT u.id
  FROM users u
  WHERE u.organization_id = w.organization_id
  ORDER BY u.created_at ASC
  LIMIT 1
)
WHERE w.created_by IS NULL;

commit;
