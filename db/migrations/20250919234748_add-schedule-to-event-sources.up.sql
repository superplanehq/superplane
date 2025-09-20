BEGIN;

ALTER TABLE event_sources
ADD COLUMN schedule jsonb,
ADD COLUMN last_triggered_at timestamp,
ADD COLUMN next_trigger_at timestamp;

CREATE INDEX idx_event_sources_next_trigger_at ON event_sources USING btree (next_trigger_at);

COMMIT;