BEGIN;

ALTER TABLE organizations DROP COLUMN display_name;
ALTER TABLE organizations DROP CONSTRAINT organizations_name_key;

COMMIT;