BEGIN;

CREATE TABLE integrations (
  id          uuid NOT NULL DEFAULT uuid_generate_v4(),
  name        CHARACTER VARYING(128) NOT NULL,
  domain_type CHARACTER VARYING(64) NOT NULL,
  domain_id   uuid NOT NULL,
  created_at  TIMESTAMP NOT NULL,
  created_by  uuid NOT NULL,
  updated_at  TIMESTAMP,
  type        CHARACTER VARYING(64) NOT NULL,
  url         CHARACTER VARYING(256) NOT NULL,
  auth_type   CHARACTER VARYING(64) NOT NULL,
  auth        jsonb NOT NULL DEFAULT '{}',
  oidc        jsonb NOT NULL DEFAULT '{}',

  PRIMARY KEY (id),
  UNIQUE (domain_type, domain_id, name)
);

CREATE TABLE resources (
  id             uuid NOT NULL DEFAULT uuid_generate_v4(),
  external_id    CHARACTER VARYING(128) NOT NULL,
  type           CHARACTER VARYING(64) NOT NULL,
  name           CHARACTER VARYING(128) NOT NULL,
  integration_id uuid NOT NULL,
  created_at     TIMESTAMP NOT NULL,
  updated_at     TIMESTAMP,

  PRIMARY KEY (id),
  FOREIGN KEY (integration_id) REFERENCES integrations(id)
);

ALTER TABLE event_sources
  ADD COLUMN resource_id uuid,
  ADD COLUMN state CHARACTER VARYING(64) NOT NULL,
  ADD FOREIGN KEY (resource_id) REFERENCES resources(id);

ALTER TABLE stages DROP COLUMN executor_spec;

CREATE TABLE stage_executors (
  id          uuid NOT NULL DEFAULT uuid_generate_v4(),
  stage_id    uuid NOT NULL,
  resource_id uuid NOT NULL,
  type        CHARACTER VARYING(64) NOT NULL,
  spec        jsonb NOT NULL DEFAULT '{}',

  PRIMARY KEY (id),
  FOREIGN KEY (stage_id) REFERENCES stages(id),
  FOREIGN KEY (resource_id) REFERENCES resources(id)
);

COMMIT;