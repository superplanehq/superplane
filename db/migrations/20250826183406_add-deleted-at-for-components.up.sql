BEGIN;

ALTER TABLE stages ADD COLUMN deleted_at timestamp with time zone;
ALTER TABLE event_sources ADD COLUMN deleted_at timestamp with time zone;
ALTER TABLE connection_groups ADD COLUMN deleted_at timestamp with time zone;

CREATE INDEX idx_stages_deleted_at ON stages (deleted_at);
CREATE INDEX idx_event_sources_deleted_at ON event_sources (deleted_at);
CREATE INDEX idx_connection_groups_deleted_at ON connection_groups (deleted_at);

COMMIT;