BEGIN;

-- Add slug column to organizations table
ALTER TABLE organizations ADD COLUMN slug VARCHAR(255);

-- Create unique index on slug (where not deleted)
CREATE UNIQUE INDEX idx_organizations_slug ON organizations(slug) WHERE deleted_at IS NULL;

-- Backfill slugs for existing organizations
UPDATE organizations 
SET slug = LOWER(REGEXP_REPLACE(name, '[^a-z0-9]+', '-', 'g')) || '-' || SUBSTRING(MD5(RANDOM()::TEXT || id::TEXT) FROM 1 FOR 6)
WHERE slug IS NULL AND deleted_at IS NULL;

-- Make slug NOT NULL for active organizations
ALTER TABLE organizations ALTER COLUMN slug SET NOT NULL;

COMMIT;