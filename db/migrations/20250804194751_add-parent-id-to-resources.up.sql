BEGIN;

ALTER TABLE resources ADD COLUMN parent_id uuid;

COMMIT;
