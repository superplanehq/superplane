-- New workspaces start with the Clarity check off. Existing rows keep
-- their stored value.
ALTER TABLE factories
  ALTER COLUMN planning_clarity SET DEFAULT FALSE;
