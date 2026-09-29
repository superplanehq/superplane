BEGIN;

WITH version_parts AS (
  SELECT
    version.id,
    version.nodes,
    version.edges,
    ARRAY(
      SELECT node ->> 'id'
      FROM jsonb_array_elements(version.nodes) AS node
      WHERE node -> 'ref' -> 'component' ->> 'name' = 'setWorkOrderStatusNote'
    ) AS removed_node_ids
  FROM workflow_versions AS version
  WHERE EXISTS (
    SELECT 1
    FROM jsonb_array_elements(version.nodes) AS node
    WHERE node -> 'ref' -> 'component' ->> 'name' = 'setWorkOrderStatusNote'
  )
), cleaned_versions AS (
  SELECT
    version.id,
    COALESCE((
      SELECT jsonb_agg(node ORDER BY position)
      FROM jsonb_array_elements(version.nodes) WITH ORDINALITY AS entries(node, position)
      WHERE NOT (node ->> 'id' = ANY (version.removed_node_ids))
    ), '[]'::jsonb) AS nodes,
    COALESCE((
      SELECT jsonb_agg(edge ORDER BY position)
      FROM (
        SELECT edge, position::numeric
        FROM jsonb_array_elements(version.edges) WITH ORDINALITY AS entries(edge, position)
        WHERE NOT (
          edge ->> 'source_id' = ANY (version.removed_node_ids)
          OR edge ->> 'target_id' = ANY (version.removed_node_ids)
        )

        UNION ALL

        SELECT
          jsonb_build_object(
            'source_id', reconnected.source_id,
            'target_id', reconnected.target_id,
            'channel', reconnected.channel
          ),
          MIN(reconnected.position)::numeric + 0.5
        FROM (
          WITH RECURSIVE paths(source_id, target_id, channel, position, visited) AS (
            SELECT
              incoming.edge ->> 'source_id',
              incoming.edge ->> 'target_id',
              incoming.edge ->> 'channel',
              incoming.position,
              ARRAY[incoming.edge ->> 'target_id']
            FROM jsonb_array_elements(version.edges) WITH ORDINALITY AS incoming(edge, position)
            WHERE incoming.edge ->> 'target_id' = ANY (version.removed_node_ids)
              AND NOT (incoming.edge ->> 'source_id' = ANY (version.removed_node_ids))

            UNION ALL

            SELECT
              path.source_id,
              outgoing.edge ->> 'target_id',
              path.channel,
              path.position,
              path.visited || (outgoing.edge ->> 'target_id')
            FROM paths AS path
            JOIN jsonb_array_elements(version.edges) AS outgoing(edge)
              ON outgoing.edge ->> 'source_id' = path.target_id
              AND outgoing.edge ->> 'channel' = 'default'
            WHERE path.target_id = ANY (version.removed_node_ids)
              AND NOT (outgoing.edge ->> 'target_id' = ANY (path.visited))
          )
          SELECT source_id, target_id, channel, position
          FROM paths
          WHERE NOT (target_id = ANY (version.removed_node_ids))
        ) AS reconnected
        WHERE NOT EXISTS (
          SELECT 1
          FROM jsonb_array_elements(version.edges) AS existing(edge)
          WHERE existing.edge ->> 'source_id' = reconnected.source_id
            AND existing.edge ->> 'target_id' = reconnected.target_id
            AND existing.edge ->> 'channel' = reconnected.channel
        )
        GROUP BY
          reconnected.source_id,
          reconnected.target_id,
          reconnected.channel
      ) AS retained_and_reconnected_edges(edge, position)
    ), '[]'::jsonb) AS edges
  FROM version_parts AS version
)
UPDATE workflow_versions AS version
SET
  nodes = cleaned.nodes,
  edges = cleaned.edges
FROM cleaned_versions AS cleaned
WHERE version.id = cleaned.id;

DELETE FROM workflow_staged_files
WHERE path = 'canvas.yaml'
  AND content LIKE '%setWorkOrderStatusNote%';

UPDATE workflow_node_executions AS execution
SET
  state = 'finished',
  result = 'cancelled',
  cancelled_at = COALESCE(execution.cancelled_at, NOW()),
  updated_at = NOW()
FROM workflow_nodes AS node
WHERE execution.workflow_id = node.workflow_id
  AND execution.node_id = node.node_id
  AND node.ref -> 'component' ->> 'name' = 'setWorkOrderStatusNote'
  AND execution.state IN ('pending', 'started', 'cancelling');

UPDATE workflow_node_requests AS request
SET
  state = 'completed',
  updated_at = NOW()
FROM workflow_nodes AS node
WHERE request.workflow_id = node.workflow_id
  AND request.node_id = node.node_id
  AND node.ref -> 'component' ->> 'name' = 'setWorkOrderStatusNote'
  AND request.state = 'pending';

DELETE FROM workflow_node_queue_items AS item
USING workflow_nodes AS node
WHERE item.workflow_id = node.workflow_id
  AND item.node_id = node.node_id
  AND node.ref -> 'component' ->> 'name' = 'setWorkOrderStatusNote';

UPDATE workflow_nodes
SET
  deleted_at = NOW(),
  updated_at = NOW()
WHERE ref -> 'component' ->> 'name' = 'setWorkOrderStatusNote'
  AND deleted_at IS NULL;

COMMIT;
