begin;

CREATE TABLE stage_connections (
  id              uuid NOT NULL DEFAULT uuid_generate_v4(),
  stage_id        uuid NOT NULL,
  source_id       uuid NOT NULL,
  source_name     CHARACTER VARYING(128) NOT NULL,
  source_type     CHARACTER VARYING(64) NOT NULL,
  filter_operator CHARACTER VARYING(16) NOT NULL,
  filters         jsonb NOT NULL,

  PRIMARY KEY (id),
  UNIQUE (stage_id, source_id),
  FOREIGN KEY (stage_id) REFERENCES stages(id)
);

CREATE INDEX uix_stage_connections_stage ON stage_connections USING btree (stage_id);

commit;
