begin;

ALTER TABLE stages ADD COLUMN secrets jsonb NOT NULL DEFAULT '[]';

CREATE TABLE secrets (
  id         uuid NOT NULL DEFAULT uuid_generate_v4(),
  canvas_id  uuid NOT NULL,
  name       CHARACTER VARYING(128) NOT NULL,
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL,
  created_by uuid NOT NULL,
  provider   CHARACTER VARYING(64) NOT NULL,
  data       bytea NOT NULL,

  PRIMARY KEY (id),
  UNIQUE (canvas_id, name),
  FOREIGN KEY (canvas_id) REFERENCES canvases(id)
);

commit;