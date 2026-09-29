BEGIN;

ALTER TABLE factories
  ADD COLUMN planning_auto_start_line_id UUID REFERENCES factory_lines(id) ON DELETE SET NULL;

COMMIT;
