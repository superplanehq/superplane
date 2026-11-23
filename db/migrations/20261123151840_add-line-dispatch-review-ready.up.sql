ALTER TABLE factory_work_order_line_dispatches
  ADD COLUMN review_ready boolean NOT NULL DEFAULT false;

-- Use traversal outcomes and snapshots, never the current board layout.
WITH latest_executions AS (
  SELECT DISTINCT ON (line_dispatch_id, step_index)
    line_dispatch_id, step_index, status, result, run_id
  FROM factory_work_order_executions
  ORDER BY line_dispatch_id, step_index, created_at DESC, id DESC
)
UPDATE factory_work_order_line_dispatches AS dispatch
SET review_ready = true
WHERE dispatch.state = 'finished' AND dispatch.result = 'passed'
  AND NOT EXISTS (
    SELECT 1 FROM factory_work_order_executions AS execution
    WHERE execution.line_dispatch_id = dispatch.id
      AND execution.status IN ('pending', 'running')
  )
  AND NOT EXISTS (
    SELECT 1 FROM factory_work_order_queue_items AS item
    WHERE item.line_dispatch_id = dispatch.id
  )
  AND (
    -- Direct handoffs record one passed step without a canvas run.
    (SELECT count(*) = 1 AND bool_and(run_id IS NULL AND status = 'finished' AND result = 'passed')
     FROM latest_executions WHERE line_dispatch_id = dispatch.id)
    OR (
      jsonb_array_length(dispatch.steps) > 0
      AND NOT EXISTS (
        SELECT 1
        FROM jsonb_array_elements(dispatch.steps) WITH ORDINALITY AS step(value, position)
        WHERE (
          position = jsonb_array_length(dispatch.steps)
          OR (COALESCE(value->>'stage_key', '') <> ''
            AND value->>'stage_key' = dispatch.steps->-1->>'stage_key')
        )
        AND NOT EXISTS (
          SELECT 1 FROM latest_executions AS execution
          WHERE execution.line_dispatch_id = dispatch.id
            AND execution.step_index = position - 1
            AND execution.status = 'finished' AND execution.result = 'passed'
        )
      )
    )
  );
