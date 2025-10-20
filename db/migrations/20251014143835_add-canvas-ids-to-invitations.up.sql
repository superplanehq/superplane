BEGIN;

ALTER TABLE organization_invitations ADD COLUMN canvas_ids jsonb DEFAULT '[]';

COMMIT;
