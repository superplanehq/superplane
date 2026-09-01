BEGIN;

-- URL-friendly, unique identifier for an organization. The frontend routes
-- by slug instead of the organization UUID, so every organization needs a
-- stable, human-readable slug. New organizations get one from
-- GenerateUniqueOrganizationSlug (pkg/models); this migration backfills
-- existing rows the same way: lowercase, ASCII-only, dashes for separators,
-- deduplicated with a numeric suffix, and never a reserved app path segment.

ALTER TABLE organizations ADD COLUMN slug TEXT NOT NULL DEFAULT '';

WITH reserved(word) AS (
  VALUES
    ('admin'), ('login'), ('signup'), ('welcome'), ('create'),
    ('setup'), ('invite'), ('install'),
    ('api'), ('health'), ('assets'), ('logout')
),
base_slugs AS (
  SELECT
    id,
    created_at,
    NULLIF(
      trim(both '-' from regexp_replace(lower(name), '[^a-z0-9]+', '-', 'g')),
      ''
    ) AS base_slug
  FROM organizations
),
normalized AS (
  SELECT
    id,
    created_at,
    COALESCE(left(base_slug, 63), 'org') AS base_slug
  FROM base_slugs
),
deduplicated AS (
  SELECT
    id,
    base_slug,
    row_number() OVER (PARTITION BY base_slug ORDER BY created_at, id) AS occurrence
  FROM normalized
)
UPDATE organizations o
SET slug = CASE
  WHEN d.occurrence = 1 AND d.base_slug NOT IN (SELECT word FROM reserved) THEN d.base_slug
  ELSE d.base_slug || '-' || (d.occurrence + (SELECT count(*) FROM reserved WHERE word = d.base_slug))::text
END
FROM deduplicated d
WHERE o.id = d.id;

ALTER TABLE organizations ALTER COLUMN slug DROP DEFAULT;

-- Ignore soft-deleted rows so a deleted organization's slug can be reused,
-- matching the existing soft-delete convention on this table.
CREATE UNIQUE INDEX organizations_slug_active_key
  ON organizations (slug)
  WHERE deleted_at IS NULL;

COMMIT;
