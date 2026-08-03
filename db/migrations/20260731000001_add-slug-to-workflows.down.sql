BEGIN;

-- Remove unique index on slug
DROP INDEX IF EXISTS idx_workflows_organization_slug;

-- Remove slug column
ALTER TABLE workflows DROP COLUMN IF EXISTS slug;

COMMIT;