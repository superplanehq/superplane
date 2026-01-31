BEGIN;

ALTER TABLE app_installation_requests ADD COLUMN spec JSONB;

COMMIT;
