ALTER TABLE factories
  ADD COLUMN planning_auto_start BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN planning_auto_start_line TEXT NOT NULL DEFAULT '';
