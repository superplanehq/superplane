BEGIN;

-- Add slug column to workflows table
ALTER TABLE workflows ADD COLUMN slug VARCHAR(255);

-- Create unique index on slug per organization (where not deleted)
CREATE UNIQUE INDEX idx_workflows_organization_slug ON workflows(organization_id, slug) WHERE deleted_at IS NULL;

-- Backfill slugs for existing workflows
UPDATE workflows 
SET slug = LOWER(REGEXP_REPLACE(name, '[^a-z0-9]+', '-', 'g')) || '-' || SUBSTRING(MD5(RANDOM()::TEXT || id::TEXT) FROM 1 FOR 6)
WHERE slug IS NULL AND deleted_at IS NULL;

-- Make slug NOT NULL for active workflows
ALTER TABLE workflows ALTER COLUMN slug SET NOT NULL;

COMMIT;