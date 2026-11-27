-- Older lines can derive their stages from steps. Persist that same layout
-- before assigning public IDs, so export never generates identities.
DO $$
DECLARE
  line record;
  stage jsonb;
  derived_stages jsonb;
  next_stages jsonb;
  base text;
  candidate text;
  suffix integer;
  step_count integer;
BEGIN
  FOR line IN
    SELECT id, factory_id, steps FROM factory_lines
    WHERE stages = '[]'::jsonb ORDER BY factory_id, created_at, id
  LOOP
    step_count := COALESCE(jsonb_array_length(NULLIF(line.steps, 'null'::jsonb)), 0);
    SELECT jsonb_agg(ordered.stage ORDER BY ordered.position) INTO derived_stages
    FROM (
      SELECT 0::bigint AS position, jsonb_build_object(
        'id', uuid_generate_v4(), 'key', 'backlog', 'name', 'Backlog',
        'kind', 'backlog', 'auto_advance', true
      ) AS stage
      UNION ALL
      SELECT step.ordinality, jsonb_build_object(
        'id', uuid_generate_v4(), 'key', 'phase-' || (step.ordinality - 1),
        'name', COALESCE(NULLIF(BTRIM(canvas.name), ''), 'Step'),
        'kind', 'phase', 'step_index', step.ordinality - 1, 'auto_advance', true
      )
      FROM jsonb_array_elements(COALESCE(NULLIF(line.steps, 'null'::jsonb), '[]'::jsonb))
        WITH ORDINALITY AS step(value, ordinality)
      LEFT JOIN workflows canvas ON canvas.id = (step.value->>'app_id')::uuid
      UNION ALL
      SELECT step_count + 1, jsonb_build_object(
        'id', uuid_generate_v4(), 'key', 'verify', 'name', 'Verify',
        'kind', 'phase', 'auto_advance', false
      )
      UNION ALL
      SELECT step_count + 2, jsonb_build_object(
        'id', uuid_generate_v4(), 'key', 'done', 'name', 'Done',
        'kind', 'done', 'auto_advance', true
      )
    ) ordered;

    next_stages := '[]'::jsonb;
    FOR stage IN SELECT value FROM jsonb_array_elements(derived_stages) LOOP
      base := COALESCE(NULLIF(btrim(left(regexp_replace(lower(stage->>'name'), '[^a-z0-9]+', '-', 'g'), 63), '-'), ''), 'stage');
      candidate := base;
      suffix := 1;
      WHILE EXISTS (
        SELECT 1 FROM factory_lines other, jsonb_array_elements(other.stages) item
        WHERE other.factory_id = line.factory_id AND item->>'configuration_id' = candidate
      ) OR EXISTS (SELECT 1 FROM jsonb_array_elements(next_stages) item WHERE item->>'configuration_id' = candidate) LOOP
        suffix := suffix + 1;
        candidate := base || '-' || suffix;
      END LOOP;
      next_stages := next_stages || jsonb_build_array(jsonb_set(stage, '{configuration_id}', to_jsonb(candidate)));
    END LOOP;
    UPDATE factory_lines SET stages = next_stages WHERE id = line.id;
  END LOOP;
END $$;
