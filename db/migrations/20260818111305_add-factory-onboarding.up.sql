BEGIN;

--
-- Persist workspace onboarding progress on factories.
-- NULL onboarding_completed_at means the wizard is still open.
-- Existing workspaces are treated as already onboarded.
--
ALTER TABLE factories
  ADD COLUMN onboarding_completed_at TIMESTAMPTZ,
  ADD COLUMN onboarding_config JSONB NOT NULL DEFAULT '{}'::jsonb;

UPDATE factories
SET onboarding_completed_at = created_at
WHERE onboarding_completed_at IS NULL;

COMMIT;
