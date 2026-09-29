BEGIN;

ALTER TABLE sentry_app_install_grants
  ADD COLUMN claimed_integration_id TEXT;

COMMIT;
