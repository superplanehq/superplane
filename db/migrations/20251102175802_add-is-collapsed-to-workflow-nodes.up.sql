BEGIN;

ALTER TABLE workflow_nodes ADD COLUMN is_collapsed boolean NOT NULL DEFAULT false;

COMMIT;