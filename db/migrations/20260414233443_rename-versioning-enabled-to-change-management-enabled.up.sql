BEGIN;

ALTER TABLE workflows DROP COLUMN versioning_enabled;
ALTER TABLE organizations DROP COLUMN versioning_enabled;

ALTER TABLE organizations ADD COLUMN change_management_enabled boolean NOT NULL DEFAULT false;
ALTER TABLE workflows ADD COLUMN change_management_enabled boolean NOT NULL DEFAULT false;

COMMIT;