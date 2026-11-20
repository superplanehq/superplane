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
