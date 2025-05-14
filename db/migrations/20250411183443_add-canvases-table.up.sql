begin;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE canvases (
  id              uuid NOT NULL DEFAULT uuid_generate_v4(),
  organization_id uuid NOT NULL,
  name            CHARACTER VARYING(128) NOT NULL,
  created_at      TIMESTAMP NOT NULL,
  created_by      uuid NOT NULL,
  updated_at      TIMESTAMP NOT NULL,

  PRIMARY KEY (id),
  UNIQUE (organization_id, name)
);

CREATE INDEX uix_canvases_orgs ON canvases USING btree (organization_id);

commit;
