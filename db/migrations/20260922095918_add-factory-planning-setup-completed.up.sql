BEGIN;

ALTER TABLE factories
  ADD COLUMN IF NOT EXISTS planning_setup_completed BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE factories
SET planning_setup_completed = TRUE
WHERE onboarding_completed_at IS NOT NULL;

COMMIT;
