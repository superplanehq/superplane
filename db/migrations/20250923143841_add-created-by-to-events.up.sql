begin;

ALTER TABLE events ADD COLUMN created_by uuid;

commit;