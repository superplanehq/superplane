begin;

ALTER TABLE stage_events ADD COLUMN cancelled_by uuid, ADD COLUMN cancelled_at TIMESTAMP;

commit;