ALTER TABLE factory_lines
  ADD COLUMN stages JSONB NOT NULL DEFAULT '[]'::jsonb;

UPDATE factory_lines AS line
SET stages = (
  SELECT jsonb_agg(stage ORDER BY position)
  FROM (
    SELECT
      0::bigint AS position,
      jsonb_build_object(
        'id', uuid_generate_v4(),
        'key', 'backlog',
        'name', 'Backlog',
        'kind', 'backlog'
      ) AS stage

    UNION ALL

    SELECT
      step.ordinality,
      jsonb_build_object(
        'id', uuid_generate_v4(),
        'key', 'phase-' || (step.ordinality - 1),
        'name', COALESCE(NULLIF(BTRIM(canvas.name), ''), 'Step'),
        'kind', 'phase',
        'step_index', step.ordinality - 1
      )
    FROM jsonb_array_elements(line.steps) WITH ORDINALITY AS step(value, ordinality)
    LEFT JOIN workflows AS canvas ON canvas.id = (step.value ->> 'app_id')::uuid

    UNION ALL

    SELECT
      jsonb_array_length(line.steps) + 1,
      jsonb_build_object(
        'id', uuid_generate_v4(),
        'key', 'verify',
        'name', 'Verify',
        'kind', 'verify'
      )

    UNION ALL

    SELECT
      jsonb_array_length(line.steps) + 2,
      jsonb_build_object(
        'id', uuid_generate_v4(),
        'key', 'done',
        'name', 'Done',
        'kind', 'done'
      )
  ) AS ordered_stages
);

ALTER TABLE factory_work_order_line_dispatches
  ADD COLUMN stages JSONB NOT NULL DEFAULT '[]'::jsonb;

ALTER TABLE factory_work_order_queue_items
  ADD COLUMN app_id uuid,
  ADD COLUMN stage_key text NOT NULL DEFAULT '';

UPDATE factory_work_order_queue_items AS queue
SET
  app_id = (dispatch.steps -> queue.step_index ->> 'app_id')::uuid,
  stage_key = COALESCE(dispatch.steps -> queue.step_index ->> 'stage_key', '')
FROM factory_work_order_line_dispatches AS dispatch
WHERE dispatch.id = queue.line_dispatch_id;

ALTER TABLE factory_work_order_queue_items
  ALTER COLUMN app_id SET NOT NULL;

DROP INDEX idx_factory_work_order_queue_items_step;

CREATE INDEX idx_factory_work_order_queue_items_stage
  ON factory_work_order_queue_items (line_id, app_id, stage_key, created_at);

ALTER TABLE factory_work_order_queue_items
  DROP CONSTRAINT factory_work_order_queue_items_line_dispatch_id_key;

ALTER TABLE factory_work_order_queue_items
  ADD CONSTRAINT factory_work_order_queue_items_line_dispatch_step_key UNIQUE (line_dispatch_id, step_index);
