begin;

ALTER TABLE stage_events ADD COLUMN inputs jsonb NOT NULL DEFAULT '{}';

commit;
