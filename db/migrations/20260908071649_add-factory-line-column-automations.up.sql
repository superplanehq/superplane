BEGIN;

--
-- Persist canvases attached to board columns. Keyed by column key
-- ("backlog", "phase-<step index>", "verify", "done") with a list of
-- factory-owned canvas ids. These canvases run on their own triggers.
-- The line does not dispatch them.
--
ALTER TABLE factory_lines
  ADD COLUMN column_automations JSONB NOT NULL DEFAULT '{}'::jsonb;

COMMIT;
