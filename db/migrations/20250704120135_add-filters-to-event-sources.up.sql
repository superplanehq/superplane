BEGIN;

ALTER TABLE event_sources
  ADD COLUMN filters jsonb NOT NULL DEFAULT '[]',
  ADD COLUMN filter_operator CHARACTER VARYING(16);

COMMIT;
