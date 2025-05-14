begin;

CREATE TABLE stage_event_tags (
  name           CHARACTER VARYING(64) NOT NULL,
  value          CHARACTER VARYING(128) NOT NULL,
  stage_event_id uuid NOT NULL,
  state          CHARACTER VARYING(64) NOT NULL,

  PRIMARY KEY (name, value, stage_event_id),
  FOREIGN KEY (stage_event_id) REFERENCES stage_events(id)
);

commit;
