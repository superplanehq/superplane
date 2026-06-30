BEGIN;

CREATE OR REPLACE FUNCTION rename_ec2_create_instance_default_edges(input_nodes jsonb, input_edges jsonb)
RETURNS jsonb
LANGUAGE sql
AS $$
  WITH ec2_create_instance_nodes AS (
    SELECT node->>'id' AS node_id
    FROM jsonb_array_elements(input_nodes) AS nodes(node)
    WHERE node->>'type' = 'component'
      AND node->'ref'->'component'->>'name' = 'aws.ec2.createInstance'
      AND COALESCE(node->>'id', '') <> ''
  )
  SELECT COALESCE(
    jsonb_agg(
      CASE
        WHEN COALESCE(edge->>'channel', 'default') IN ('', 'default')
          AND COALESCE(edge->>'source_id', edge->>'sourceId') IN (
            SELECT node_id FROM ec2_create_instance_nodes
          )
          THEN jsonb_set(edge, '{channel}', to_jsonb('created'::text), true)
        ELSE edge
      END
      ORDER BY edges.ordinality
    ),
    '[]'::jsonb
  )
  FROM jsonb_array_elements(input_edges) WITH ORDINALITY AS edges(edge, ordinality);
$$;

UPDATE public.workflow_versions
SET edges = rename_ec2_create_instance_default_edges(nodes, edges)
WHERE EXISTS (
  SELECT 1
  FROM jsonb_array_elements(nodes) AS node(item)
  WHERE item->>'type' = 'component'
    AND item->'ref'->'component'->>'name' = 'aws.ec2.createInstance'
)
AND EXISTS (
  SELECT 1
  FROM jsonb_array_elements(edges) AS edge(item)
  WHERE COALESCE(item->>'channel', 'default') IN ('', 'default')
);

DROP FUNCTION rename_ec2_create_instance_default_edges(jsonb, jsonb);

COMMIT;
