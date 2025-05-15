begin;

CREATE TABLE stages (
  id                uuid NOT NULL DEFAULT uuid_generate_v4(),
  name              CHARACTER VARYING(128) NOT NULL,
  canvas_id         uuid NOT NULL,
  created_at        TIMESTAMP NOT NULL,
  created_by        uuid NOT NULL,
  run_template      jsonb NOT NULL,
  conditions        jsonb,
  use               jsonb,

  PRIMARY KEY (id),
  UNIQUE (canvas_id, name),
  FOREIGN KEY (canvas_id) REFERENCES canvases(id)
);

CREATE INDEX uix_stages_canvas ON stages USING btree (canvas_id);

commit;
