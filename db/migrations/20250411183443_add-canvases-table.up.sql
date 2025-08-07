begin;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE canvases (
  id              uuid NOT NULL DEFAULT uuid_generate_v4(),
  organization_id UUID NOT NULL,
  name            CHARACTER VARYING(128) NOT NULL,
  created_at      TIMESTAMP NOT NULL,
  created_by      uuid NOT NULL,
  updated_at      TIMESTAMP NOT NULL,

  PRIMARY KEY (id),
  FOREIGN KEY (organization_id) REFERENCES organizations(id),
  UNIQUE (organization_id, name)
);

commit;
