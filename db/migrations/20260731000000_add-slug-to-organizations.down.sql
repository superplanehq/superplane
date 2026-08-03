BEGIN;

-- Remove unique index on slug
DROP INDEX IF EXISTS idx_organizations_slug;

-- Remove slug column
ALTER TABLE organizations DROP COLUMN IF EXISTS slug;

COMMIT;