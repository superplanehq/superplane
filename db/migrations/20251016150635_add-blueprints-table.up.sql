begin;

CREATE TABLE blueprints (
  id              uuid NOT NULL DEFAULT uuid_generate_v4(),
  organization_id uuid NOT NULL,
  name            CHARACTER VARYING(128) NOT NULL,
  description     TEXT,
  created_at      TIMESTAMP NOT NULL,
  updated_at      TIMESTAMP NOT NULL,
  nodes           JSONB NOT NULL DEFAULT '[]'::jsonb,
  edges           JSONB NOT NULL DEFAULT '[]'::jsonb,
  configuration   JSONB NOT NULL DEFAULT '[]'::jsonb,
  output_channels JSONB NOT NULL DEFAULT '[]'::jsonb,

  PRIMARY KEY (id),
  UNIQUE (organization_id, name)
);

CREATE INDEX idx_blueprints_organization_id ON blueprints(organization_id);

commit;
