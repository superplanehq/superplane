begin;

CREATE TABLE app_installations (
  id                uuid NOT NULL DEFAULT uuid_generate_v4(),
  organization_id   uuid NOT NULL,
  app_name          CHARACTER VARYING(255) NOT NULL,
  installation_name CHARACTER VARYING(255) NOT NULL,
  state             CHARACTER VARYING(32) NOT NULL,
  configuration     JSONB NOT NULL DEFAULT '{}',
  metadata          JSONB NOT NULL DEFAULT '{}',
  browser_action    JSONB,
  created_at        TIMESTAMP NOT NULL,
  updated_at        TIMESTAMP NOT NULL,

  PRIMARY KEY (id),
  FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE
);


commit;