BEGIN;

ALTER TABLE app_installations ADD COLUMN capabilities jsonb NOT NULL DEFAULT '[]';
ALTER TABLE app_installations ADD COLUMN properties JSONB NOT NULL DEFAULT '[]';
ALTER TABLE app_installations ADD COLUMN setup_state JSONB;
ALTER TABLE app_installation_secrets ADD COLUMN editable BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE app_installation_secrets ADD COLUMN label TEXT;
ALTER TABLE app_installation_secrets ADD COLUMN description TEXT;

COMMIT;
