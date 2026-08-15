BEGIN;

ALTER TABLE factory_work_order_executions
  ADD COLUMN line_steps JSONB NOT NULL DEFAULT '[]'::jsonb;

-- Backfill: best-effort snapshot from today's line definition. Rows
-- backfilled this way are NOT a true historical record -- if the line
-- was edited between when the step ran and today, this reflects the
-- edited version, not what actually ran at the time.
UPDATE factory_work_order_executions e
SET line_steps = COALESCE(l.steps, '[]'::jsonb)
FROM factory_lines l
WHERE l.id = e.line_id;

COMMIT;
