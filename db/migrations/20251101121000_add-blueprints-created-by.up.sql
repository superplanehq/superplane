begin;

ALTER TABLE blueprints
  ADD COLUMN IF NOT EXISTS created_by uuid;

-- Backfill using the first user in the organization
UPDATE blueprints b
SET created_by = (
  SELECT u.id
  FROM users u
  WHERE u.organization_id = b.organization_id
  ORDER BY u.created_at ASC
  LIMIT 1
)
WHERE b.created_by IS NULL;

commit;

