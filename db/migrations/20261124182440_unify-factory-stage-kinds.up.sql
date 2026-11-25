-- Preserve stage settings and order, including finished retry snapshots.
UPDATE factory_lines AS line
SET stages = normalized.stages
FROM (
  SELECT id, jsonb_agg(
    CASE
      WHEN stage->>'kind' = 'verify' THEN
        jsonb_build_object('auto_advance', false) || stage || '{"kind":"phase"}'::jsonb
      WHEN stage->>'kind' = 'custom' THEN stage || '{"kind":"phase"}'::jsonb
      ELSE stage
    END ORDER BY position
  ) AS stages
  FROM factory_lines,
    jsonb_array_elements(CASE WHEN jsonb_typeof(stages) = 'array' THEN stages ELSE '[]'::jsonb END) WITH ORDINALITY AS element(stage, position)
  GROUP BY id
) AS normalized
WHERE line.id = normalized.id;

UPDATE factory_work_order_line_dispatches AS dispatch
SET stages = normalized.stages
FROM (
  SELECT id, jsonb_agg(
    CASE
      WHEN stage->>'kind' = 'verify' THEN
        jsonb_build_object('auto_advance', false) || stage || '{"kind":"phase"}'::jsonb
      WHEN stage->>'kind' = 'custom' THEN stage || '{"kind":"phase"}'::jsonb
      ELSE stage
    END ORDER BY position
  ) AS stages
  FROM factory_work_order_line_dispatches,
    jsonb_array_elements(CASE WHEN jsonb_typeof(stages) = 'array' THEN stages ELSE '[]'::jsonb END) WITH ORDINALITY AS element(stage, position)
  GROUP BY id
) AS normalized
WHERE dispatch.id = normalized.id;

UPDATE factory_lines AS line
SET steps = normalized.steps
FROM (
  SELECT id, jsonb_agg(
    CASE WHEN step->>'stage_kind' IN ('custom', 'verify')
      THEN step || '{"stage_kind":"phase"}'::jsonb ELSE step END
    ORDER BY position
  ) AS steps
  FROM factory_lines,
    jsonb_array_elements(CASE WHEN jsonb_typeof(steps) = 'array' THEN steps ELSE '[]'::jsonb END) WITH ORDINALITY AS element(step, position)
  GROUP BY id
) AS normalized
WHERE line.id = normalized.id;

UPDATE factory_work_order_line_dispatches AS dispatch
SET steps = normalized.steps
FROM (
  SELECT id, jsonb_agg(
    CASE WHEN step->>'stage_kind' IN ('custom', 'verify')
      THEN step || '{"stage_kind":"phase"}'::jsonb ELSE step END
    ORDER BY position
  ) AS steps
  FROM factory_work_order_line_dispatches,
    jsonb_array_elements(CASE WHEN jsonb_typeof(steps) = 'array' THEN steps ELSE '[]'::jsonb END) WITH ORDINALITY AS element(step, position)
  GROUP BY id
) AS normalized
WHERE dispatch.id = normalized.id;
