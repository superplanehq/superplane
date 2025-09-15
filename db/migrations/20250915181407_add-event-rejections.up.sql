BEGIN;

CREATE TABLE event_rejections (
  id             uuid NOT NULL DEFAULT uuid_generate_v4(),
  event_id       uuid NOT NULL,
  target_type    CHARACTER VARYING(64) NOT NULL,
  target_id      uuid NOT NULL,
  reason         CHARACTER VARYING(64) NOT NULL,
  message        TEXT,
  rejected_at    TIMESTAMP NOT NULL DEFAULT NOW(),

  PRIMARY KEY (id),
  FOREIGN KEY (event_id) REFERENCES events(id)
);

CREATE INDEX idx_event_rejections_event_id ON event_rejections(event_id);
CREATE INDEX idx_event_rejections_target ON event_rejections(target_type, target_id);

COMMIT;