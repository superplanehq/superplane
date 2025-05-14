begin;

CREATE TABLE event_sources (
  id              uuid NOT NULL DEFAULT uuid_generate_v4(),
  organization_id uuid NOT NULL,
  canvas_id       uuid NOT NULL,
  name            CHARACTER VARYING(128) NOT NULL,
  created_at      TIMESTAMP NOT NULL,
  updated_at      TIMESTAMP NOT NULL,
  key             BYTEA NOT NULL,

  PRIMARY KEY (id),
  UNIQUE (organization_id, canvas_id, name),
  FOREIGN KEY (canvas_id) REFERENCES canvases(id)
);

CREATE INDEX uix_event_sources_canvas ON event_sources USING btree (canvas_id);

commit;
