BEGIN;

ALTER TABLE stage_events RENAME COLUMN cancelled_by TO discarded_by;
ALTER TABLE stage_events RENAME COLUMN cancelled_at TO discarded_at;

COMMIT;
