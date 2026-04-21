BEGIN;

CREATE OR REPLACE FUNCTION flatten_group_widget_nodes(input_nodes jsonb)
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
    FROM jsonb_array_elements(input_nodes) WITH ORDINALITY AS items(node, ordinality)
    WHERE node->>'type' = 'widget'
      AND node->'ref'->'widget'->>'name' = 'group'
      AND COALESCE(node->>'id', '') <> ''
  ),
  direct_group_children AS (
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
  inherited_offsets AS (
    SELECT
      child_id AS node_id,
      group_id AS parent_group_id,
      group_x AS offset_x,
      group_y AS offset_y,
      ARRAY[group_id, child_id] AS visited_ids,
      1 AS depth
    FROM direct_group_children

    UNION ALL

    SELECT
      inherited_offsets.node_id,
      parent.group_id AS parent_group_id,
      inherited_offsets.offset_x + parent.group_x AS offset_x,
      inherited_offsets.offset_y + parent.group_y AS offset_y,
      inherited_offsets.visited_ids || parent.group_id,
      inherited_offsets.depth + 1 AS depth
    FROM inherited_offsets
    JOIN direct_group_children AS parent
      ON inherited_offsets.parent_group_id = parent.child_id
    WHERE NOT parent.group_id = ANY(inherited_offsets.visited_ids)
  ),
  resolved_offsets AS (
    SELECT DISTINCT ON (node_id)
      node_id,
      offset_x,
      offset_y
    FROM inherited_offsets
    ORDER BY node_id, depth DESC
  )
  SELECT COALESCE(
    jsonb_agg(
      CASE
        WHEN resolved_offsets.node_id IS NULL THEN items.node
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
    ON resolved_offsets.node_id = items.node->>'id'
  WHERE NOT (
    items.node->>'type' = 'widget'
    AND items.node->'ref'->'widget'->>'name' = 'group'
  );
$$;

UPDATE public.workflow_versions
SET nodes = flatten_group_widget_nodes(nodes)
WHERE EXISTS (
  SELECT 1
  FROM jsonb_array_elements(nodes) AS node(item)
  WHERE item->>'type' = 'widget'
    AND item->'ref'->'widget'->>'name' = 'group'
);

DROP FUNCTION flatten_group_widget_nodes(jsonb);

COMMIT;
