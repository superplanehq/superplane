BEGIN;

ALTER TABLE organization_invitations ADD COLUMN canvas_ids uuid[];

COMMIT;
