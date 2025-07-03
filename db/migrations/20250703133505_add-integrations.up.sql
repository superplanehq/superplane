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

COMMIT;