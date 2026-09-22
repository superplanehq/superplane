BEGIN;

-- Planning and the Confidence check are on for new workspaces. The
-- Clarity check is opt-in.
ALTER TABLE factories
  ADD COLUMN IF NOT EXISTS planning_enabled BOOLEAN NOT NULL DEFAULT TRUE,
  ADD COLUMN IF NOT EXISTS planning_clarity BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS planning_confidence BOOLEAN NOT NULL DEFAULT TRUE;

COMMIT;
