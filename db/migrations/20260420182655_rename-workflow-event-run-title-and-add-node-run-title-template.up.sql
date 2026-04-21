BEGIN;

ALTER TABLE workflow_events RENAME COLUMN custom_name TO run_title;

ALTER TABLE workflow_nodes ADD COLUMN run_title_template text;

COMMIT;
