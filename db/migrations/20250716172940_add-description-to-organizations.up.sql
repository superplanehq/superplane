BEGIN;

ALTER TABLE organizations ADD COLUMN description TEXT DEFAULT '';

COMMIT;