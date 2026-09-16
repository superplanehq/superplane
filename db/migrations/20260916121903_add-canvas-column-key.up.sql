BEGIN;

--
-- Factory event automations can be shown on the Verify or Done board
-- column. Null means the canvas is not attached to either column.
--
ALTER TABLE workflows
  ADD COLUMN column_key TEXT;

COMMIT;
