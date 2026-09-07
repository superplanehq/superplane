BEGIN;

--
-- Persist board column colors on factory lines so they survive a page
-- refresh. Keyed by column key ("backlog", "phase-<step index>", "verify",
-- "done") with a color id value; missing keys use the default color.
--
ALTER TABLE factory_lines
  ADD COLUMN column_colors JSONB NOT NULL DEFAULT '{}'::jsonb;

COMMIT;
