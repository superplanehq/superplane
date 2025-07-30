BEGIN;

ALTER TABLE canvases ADD COLUMN description TEXT;

ALTER TABLE stages ADD COLUMN description TEXT;

ALTER TABLE event_sources ADD COLUMN description TEXT;

ALTER TABLE connection_groups ADD COLUMN description TEXT;

COMMIT;