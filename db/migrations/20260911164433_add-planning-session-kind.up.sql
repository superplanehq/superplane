BEGIN;

ALTER TABLE factory_planning_sessions
  ADD COLUMN kind TEXT;

UPDATE factory_planning_sessions
SET kind = 'task_creation';

ALTER TABLE factory_planning_sessions
  ALTER COLUMN kind SET NOT NULL,
  ADD CONSTRAINT factory_planning_sessions_kind_check
    CHECK (kind IN ('task_creation', 'work_order_analysis'));

CREATE UNIQUE INDEX idx_factory_planning_sessions_analysis_work_order
  ON factory_planning_sessions (organization_id, factory_id, draft_work_order_id)
  WHERE kind = 'work_order_analysis' AND draft_work_order_id IS NOT NULL;

WITH backlog_versions AS (
  SELECT versions.id
  FROM workflow_versions AS versions
  WHERE jsonb_array_length(versions.nodes) = 5
    AND jsonb_array_length(versions.edges) = 4
    AND (
      SELECT count(*)
      FROM jsonb_array_elements(versions.nodes) AS node
      WHERE CASE node->>'id'
        WHEN 'trigger' THEN node->'ref'->'trigger'->>'name' = 'onWorkOrder'
        WHEN 'analyze' THEN node->'ref'->'component'->>'name' IN (
          'runner',
          'runnerClaudeCode',
          'runnerCodex',
          'runnerOpenRouter'
        )
        WHEN 'report-confidence' THEN node->'ref'->'component'->>'name' = 'reportWorkOrderCheck'
        WHEN 'attach-intent' THEN node->'ref'->'component'->>'name' = 'addWorkOrderArtifact'
        WHEN 'add-run-error' THEN node->'ref'->'component'->>'name' = 'addRunError'
        ELSE FALSE
      END
    ) = 5
    AND (
      SELECT count(*)
      FROM jsonb_array_elements(versions.edges) AS edge
      WHERE (
        edge->>'source_id',
        edge->>'target_id',
        edge->>'channel'
      ) IN (
        ('trigger', 'analyze', 'default'),
        ('analyze', 'report-confidence', 'passed'),
        ('analyze', 'attach-intent', 'passed'),
        ('analyze', 'add-run-error', 'failed')
      )
    ) = 4
), ranked_sessions AS (
  SELECT
    sessions.id,
    row_number() OVER (
      PARTITION BY
        sessions.organization_id,
        sessions.factory_id,
        sessions.draft_work_order_id
      ORDER BY sessions.created_at DESC, sessions.id DESC
    ) AS position
  FROM factory_planning_sessions AS sessions
  JOIN workflow_runs AS runs
    ON runs.id = sessions.canvas_run_id
  JOIN backlog_versions
    ON backlog_versions.id = runs.version_id
  WHERE sessions.kind = 'task_creation'
    AND sessions.draft_work_order_id IS NOT NULL
    AND NOT EXISTS (
      SELECT 1
      FROM factory_planning_sessions AS analysis_sessions
      WHERE analysis_sessions.organization_id = sessions.organization_id
        AND analysis_sessions.factory_id = sessions.factory_id
        AND analysis_sessions.draft_work_order_id = sessions.draft_work_order_id
        AND analysis_sessions.kind = 'work_order_analysis'
    )
)
UPDATE factory_planning_sessions
SET kind = 'work_order_analysis'
FROM ranked_sessions
WHERE factory_planning_sessions.id = ranked_sessions.id
  AND ranked_sessions.position = 1;

COMMIT;
