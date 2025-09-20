BEGIN;

ALTER TABLE event_sources ADD COLUMN schedule jsonb;

COMMIT;