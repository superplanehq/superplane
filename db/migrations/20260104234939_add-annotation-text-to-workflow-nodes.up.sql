BEGIN;

ALTER TABLE workflow_nodes ADD COLUMN annotation_text VARCHAR(5000);

COMMIT;