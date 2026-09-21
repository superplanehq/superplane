BEGIN;

WITH cleaned_versions AS (
  SELECT
    version.id,
    COALESCE((
      SELECT jsonb_agg(node ORDER BY position)
      FROM jsonb_array_elements(version.nodes) WITH ORDINALITY AS entries(node, position)
      WHERE node -> 'ref' -> 'component' ->> 'name' IS DISTINCT FROM 'setWorkOrderStatusNote'
    ), '[]'::jsonb) AS nodes,
    COALESCE((
      SELECT jsonb_agg(edge ORDER BY position)
      FROM jsonb_array_elements(version.edges) WITH ORDINALITY AS entries(edge, position)
      WHERE NOT EXISTS (
        SELECT 1
        FROM jsonb_array_elements(version.nodes) AS node
        WHERE node -> 'ref' -> 'component' ->> 'name' = 'setWorkOrderStatusNote'
          AND node ->> 'id' IN (edge ->> 'source_id', edge ->> 'target_id')
      )
    ), '[]'::jsonb) AS edges
  FROM workflow_versions AS version
  WHERE EXISTS (
    SELECT 1
    FROM jsonb_array_elements(version.nodes) AS node
    WHERE node -> 'ref' -> 'component' ->> 'name' = 'setWorkOrderStatusNote'
  )
)
UPDATE workflow_versions AS version
SET
  nodes = cleaned.nodes,
  edges = cleaned.edges
FROM cleaned_versions AS cleaned
WHERE version.id = cleaned.id;

UPDATE workflow_nodes
SET
  deleted_at = NOW(),
  updated_at = NOW()
WHERE ref -> 'component' ->> 'name' = 'setWorkOrderStatusNote'
  AND deleted_at IS NULL;

COMMIT;
