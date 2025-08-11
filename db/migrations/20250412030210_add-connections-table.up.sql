begin;

CREATE TABLE connections (
  id              uuid NOT NULL DEFAULT uuid_generate_v4(),
  canvas_id       uuid NOT NULL,
  stage_id        uuid NOT NULL,
  source_id       uuid NOT NULL,
  source_name     CHARACTER VARYING(128) NOT NULL,
  source_type     CHARACTER VARYING(64) NOT NULL,
  target_id       uuid NOT NULL,
  target_type     character varying(64) NOT NULL,
  filter_operator CHARACTER VARYING(16) NOT NULL,
  filters         jsonb NOT NULL,

  PRIMARY KEY (id),
  UNIQUE (stage_id, source_id),
  FOREIGN KEY (canvas_id) REFERENCES canvases(id),
  FOREIGN KEY (stage_id) REFERENCES stages(id)
);

commit;
