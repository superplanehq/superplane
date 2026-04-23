BEGIN;

ALTER TABLE app_installations ADD COLUMN next_setup_step JSONB;
ALTER TABLE app_installations ADD COLUMN parameters JSONB NOT NULL DEFAULT '[]';
ALTER TABLE app_installations ADD COLUMN capabilities JSONB NOT NULL DEFAULT '[]';
ALTER TABLE app_installation_secrets ADD COLUMN editable BOOLEAN NOT NULL DEFAULT FALSE;

COMMIT;