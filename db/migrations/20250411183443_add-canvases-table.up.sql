begin;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE canvases (
  id              uuid NOT NULL DEFAULT uuid_generate_v4(),
  name            CHARACTER VARYING(128) NOT NULL,
  created_at      TIMESTAMP NOT NULL,
  created_by      uuid NOT NULL,
  updated_at      TIMESTAMP NOT NULL,

  PRIMARY KEY (id),
  UNIQUE (name)
);

commit;
