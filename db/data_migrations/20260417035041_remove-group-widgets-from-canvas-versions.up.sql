BEGIN;

CREATE OR REPLACE FUNCTION normalize_canvas_nodes_without_groups(input_nodes jsonb)
RETURNS jsonb
LANGUAGE sql
AS $$
  WITH RECURSIVE group_nodes AS (
    SELECT
      node,
      ordinality,
      node->>'id' AS group_id,
      COALESCE((node->'position'->>'x')::numeric, 0) AS group_x,
      COALESCE((node->'position'->>'y')::numeric, 0) AS group_y
    FROM jsonb_array_elements(input_nodes) WITH ORDINALITY AS t(node, ordinality)
    WHERE node->>'type' = 'TYPE_WIDGET'
      AND node->'widget'->>'name' = 'group'
      AND COALESCE(node->>'id', '') <> ''
  ),
  group_children AS (
    SELECT DISTINCT ON (child_id)
      group_id,
      child_id,
      group_x,
      group_y
    FROM (
      SELECT
        group_nodes.group_id,
        child.child_id,
        group_nodes.group_x,
        group_nodes.group_y,
        group_nodes.ordinality
      FROM group_nodes
      CROSS JOIN LATERAL jsonb_array_elements_text(
        COALESCE(group_nodes.node->'configuration'->'childNodeIds', '[]'::jsonb)
      ) AS child(child_id)
      WHERE child.child_id <> ''
    ) AS source
    ORDER BY child_id, ordinality
  ),
  group_offsets AS (
    SELECT
      child_id AS descendant_id,
      group_id AS current_group_id,
      group_x AS offset_x,
      group_y AS offset_y,
      ARRAY[group_id, child_id] AS visited_path,
      1 AS depth
    FROM group_children

    UNION ALL

    SELECT
      group_offsets.descendant_id,
      parent.group_id AS current_group_id,
      group_offsets.offset_x + parent.group_x AS offset_x,
      group_offsets.offset_y + parent.group_y AS offset_y,
      group_offsets.visited_path || parent.group_id,
      group_offsets.depth + 1 AS depth
    FROM group_offsets
    JOIN group_children AS parent
      ON group_offsets.current_group_id = parent.child_id
    WHERE NOT parent.group_id = ANY(group_offsets.visited_path)
  ),
  resolved_offsets AS (
    SELECT DISTINCT ON (descendant_id)
      descendant_id,
      offset_x,
      offset_y
    FROM group_offsets
    ORDER BY descendant_id, depth DESC
  )
  SELECT COALESCE(
    jsonb_agg(
      CASE
        WHEN resolved_offsets.descendant_id IS NULL THEN items.node
        ELSE jsonb_set(
          jsonb_set(
            items.node,
            '{position,x}',
            to_jsonb(ROUND(COALESCE((items.node->'position'->>'x')::numeric, 0) + resolved_offsets.offset_x)::int),
            true
          ),
          '{position,y}',
          to_jsonb(ROUND(COALESCE((items.node->'position'->>'y')::numeric, 0) + resolved_offsets.offset_y)::int),
          true
        )
      END
      ORDER BY items.ordinality
    ),
    '[]'::jsonb
  )
  FROM jsonb_array_elements(input_nodes) WITH ORDINALITY AS items(node, ordinality)
  LEFT JOIN resolved_offsets
    ON resolved_offsets.descendant_id = items.node->>'id'
  WHERE NOT (
    items.node->>'type' = 'TYPE_WIDGET'
    AND items.node->'widget'->>'name' = 'group'
  );
$$;

UPDATE public.workflow_versions
SET nodes = normalize_canvas_nodes_without_groups(nodes)
WHERE EXISTS (
  SELECT 1
  FROM jsonb_array_elements(nodes) AS node(item)
  WHERE item->>'type' = 'TYPE_WIDGET'
    AND item->'widget'->>'name' = 'group'
);

DROP FUNCTION normalize_canvas_nodes_without_groups(jsonb);

COMMIT;
