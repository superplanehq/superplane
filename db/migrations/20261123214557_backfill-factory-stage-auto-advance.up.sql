-- Stages saved before auto-advance existed have no auto_advance key.
-- Verify and the last work stage before Done stay stopped, so tasks do not
-- move into Done automatically. Every other stage advances.

UPDATE factory_lines AS line
SET stages = backfilled.stages
FROM (
  SELECT
    stage_rows.id,
    jsonb_agg(
      CASE
        WHEN stage_rows.stage ? 'auto_advance' THEN stage_rows.stage
        ELSE stage_rows.stage || jsonb_build_object(
          'auto_advance',
          NOT (
            stage_rows.stage->>'kind' = 'verify'
            OR (stage_rows.stage->>'kind' IN ('phase', 'custom') AND stage_rows.next_kind = 'done')
          )
        )
      END
      ORDER BY stage_rows.position
    ) AS stages
  FROM (
    SELECT
      l.id,
      element.stage,
      element.position,
      lead(element.stage->>'kind') OVER (PARTITION BY l.id ORDER BY element.position) AS next_kind
    FROM factory_lines AS l,
      jsonb_array_elements(l.stages) WITH ORDINALITY AS element(stage, position)
  ) AS stage_rows
  GROUP BY stage_rows.id
) AS backfilled
WHERE line.id = backfilled.id
  AND EXISTS (
    SELECT 1 FROM jsonb_array_elements(line.stages) AS existing(stage)
    WHERE NOT existing.stage ? 'auto_advance'
  );

UPDATE factory_work_order_line_dispatches AS dispatch
SET stages = backfilled.stages
FROM (
  SELECT
    stage_rows.id,
    jsonb_agg(
      CASE
        WHEN stage_rows.stage ? 'auto_advance' THEN stage_rows.stage
        ELSE stage_rows.stage || jsonb_build_object(
          'auto_advance',
          NOT (
            stage_rows.stage->>'kind' = 'verify'
            OR (stage_rows.stage->>'kind' IN ('phase', 'custom') AND stage_rows.next_kind = 'done')
          )
        )
      END
      ORDER BY stage_rows.position
    ) AS stages
  FROM (
    SELECT
      d.id,
      element.stage,
      element.position,
      lead(element.stage->>'kind') OVER (PARTITION BY d.id ORDER BY element.position) AS next_kind
    FROM factory_work_order_line_dispatches AS d,
      jsonb_array_elements(d.stages) WITH ORDINALITY AS element(stage, position)
    WHERE d.state <> 'finished'
  ) AS stage_rows
  GROUP BY stage_rows.id
) AS backfilled
WHERE dispatch.id = backfilled.id
  AND EXISTS (
    SELECT 1 FROM jsonb_array_elements(dispatch.stages) AS existing(stage)
    WHERE NOT existing.stage ? 'auto_advance'
  );
