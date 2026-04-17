BEGIN;

UPDATE public.workflow_versions
SET nodes = COALESCE(
  (
    SELECT jsonb_agg(item ORDER BY ordinality)
    FROM jsonb_array_elements(nodes) WITH ORDINALITY AS items(item, ordinality)
    WHERE NOT (
      item->>'type' = 'TYPE_WIDGET'
      AND item->'widget'->>'name' = 'group'
    )
  ),
  '[]'::jsonb
)
WHERE EXISTS (
  SELECT 1
  FROM jsonb_array_elements(nodes) AS node(item)
  WHERE item->>'type' = 'TYPE_WIDGET'
    AND item->'widget'->>'name' = 'group'
);

COMMIT;
