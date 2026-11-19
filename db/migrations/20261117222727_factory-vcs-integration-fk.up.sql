BEGIN;

ALTER TABLE factories
    ADD COLUMN vcs_integration_id uuid REFERENCES app_installations(id) ON DELETE SET NULL;

UPDATE factories f
SET vcs_integration_id = i.id
FROM app_installations i
WHERE f.onboarding_config->>'vcs_integration_id' ~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
  AND i.id = (f.onboarding_config->>'vcs_integration_id')::uuid
  AND i.organization_id = f.organization_id
  AND i.deleted_at IS NULL;

UPDATE factories
SET onboarding_config = onboarding_config - 'vcs_integration_id'
WHERE onboarding_config ? 'vcs_integration_id';

CREATE INDEX idx_factories_vcs_integration_id ON factories (vcs_integration_id);

COMMIT;
