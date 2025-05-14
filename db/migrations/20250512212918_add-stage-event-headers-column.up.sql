begin;

ALTER TABLE events ADD COLUMN headers jsonb NOT NULL DEFAULT '{}';

commit;
