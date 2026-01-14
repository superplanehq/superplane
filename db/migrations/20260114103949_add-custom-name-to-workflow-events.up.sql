BEGIN;

ALTER TABLE workflow_events
ADD COLUMN custom_name text;

COMMIT;