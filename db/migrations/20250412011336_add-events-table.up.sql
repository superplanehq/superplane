begin;

CREATE TABLE events (
  id              uuid NOT NULL DEFAULT uuid_generate_v4(),
  source_id       uuid NOT NULL,
  source_name     CHARACTER VARYING(128) NOT NULL,
  source_type     CHARACTER VARYING(64) NOT NULL,
  received_at     TIMESTAMP NOT NULL,
  raw             jsonb NOT NULL,
  state           CHARACTER VARYING(64) NOT NULL,

  PRIMARY KEY (id)
);

CREATE INDEX uix_events_source ON events USING btree (source_id);

commit;
