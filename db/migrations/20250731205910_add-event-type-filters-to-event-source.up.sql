BEGIN;

ALTER TABLE event_sources ADD COLUMN event_types jsonb NOT NULL DEFAULT '[]';

COMMIT;
