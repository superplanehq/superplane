BEGIN;

-- Productive intakes that already filter by task list must listen for
-- task.updated so a move onto a selected list creates a SuperPlane task.
-- Webhook handling reads workflow_nodes.configuration. A later settings
-- save publishes the live version snapshot, so both must carry the actions.

WITH filtered_canvases AS (
  SELECT workflow.id AS canvas_id, workflow.live_version_id
  FROM workflows AS workflow
  JOIN factory_intakes AS intake ON intake.canvas_id = workflow.id
  JOIN workflow_versions AS live_version ON live_version.id = workflow.live_version_id
  WHERE intake.source = 'productive-tasks'
    AND workflow.deleted_at IS NULL
    AND workflow.live_version_id IS NOT NULL
    AND EXISTS (
      SELECT 1
      FROM jsonb_array_elements(live_version.nodes) AS node
      WHERE COALESCE(node -> 'configuration' ->> 'expression', '')
        LIKE '%task_list.data.id%'
    )
)
UPDATE workflow_nodes AS node
SET
  configuration = jsonb_set(
    COALESCE(node.configuration, '{}'::jsonb),
    '{actions}',
    '["created", "updated"]'::jsonb,
    true
  ),
  updated_at = NOW()
FROM filtered_canvases
WHERE node.workflow_id = filtered_canvases.canvas_id
  AND node.deleted_at IS NULL
  AND node.ref -> 'trigger' ->> 'name' = 'productive.onTask';

WITH filtered_canvases AS (
  SELECT workflow.id AS canvas_id, workflow.live_version_id
  FROM workflows AS workflow
  JOIN factory_intakes AS intake ON intake.canvas_id = workflow.id
  JOIN workflow_versions AS live_version ON live_version.id = workflow.live_version_id
  WHERE intake.source = 'productive-tasks'
    AND workflow.deleted_at IS NULL
    AND workflow.live_version_id IS NOT NULL
    AND EXISTS (
      SELECT 1
      FROM jsonb_array_elements(live_version.nodes) AS node
      WHERE COALESCE(node -> 'configuration' ->> 'expression', '')
        LIKE '%task_list.data.id%'
    )
)
UPDATE workflow_versions AS version
SET nodes = COALESCE((
  SELECT jsonb_agg(
    CASE
      WHEN elem -> 'ref' -> 'trigger' ->> 'name' = 'productive.onTask'
      THEN jsonb_set(
        jsonb_set(
          elem,
          '{configuration}',
          COALESCE(elem -> 'configuration', '{}'::jsonb),
          true
        ),
        '{configuration,actions}',
        '["created", "updated"]'::jsonb,
        true
      )
      ELSE elem
    END
    ORDER BY ordinality
  )
  FROM jsonb_array_elements(version.nodes) WITH ORDINALITY AS t(elem, ordinality)
), version.nodes)
FROM filtered_canvases
WHERE version.id = filtered_canvases.live_version_id
  AND jsonb_typeof(version.nodes) = 'array';

COMMIT;
