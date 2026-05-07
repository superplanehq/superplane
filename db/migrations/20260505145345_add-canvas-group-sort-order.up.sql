ALTER TABLE canvas_groups
  ADD COLUMN sort_order bigint NOT NULL DEFAULT 0;

WITH ranked_groups AS (
  SELECT
    id,
    ROW_NUMBER() OVER (
      PARTITION BY organization_id
      ORDER BY created_at DESC, id DESC
    ) AS sort_order
  FROM canvas_groups
)
UPDATE canvas_groups
SET sort_order = ranked_groups.sort_order
FROM ranked_groups
WHERE canvas_groups.id = ranked_groups.id;

ALTER TABLE canvas_groups
  ALTER COLUMN sort_order DROP DEFAULT;
