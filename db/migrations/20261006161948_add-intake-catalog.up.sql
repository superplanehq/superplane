BEGIN;

CREATE TABLE intake_catalog_entries (
  key varchar(64) PRIMARY KEY,
  name varchar(255) NOT NULL,
  category varchar(64) NOT NULL,
  status varchar(32) NOT NULL DEFAULT 'planned',
  status_note text NOT NULL DEFAULT '',
  enabled_for_all boolean NOT NULL DEFAULT false,
  created_at timestamp without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at timestamp without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_by uuid
);

CREATE TABLE intake_catalog_organizations (
  entry_key varchar(64) NOT NULL REFERENCES intake_catalog_entries(key) ON DELETE CASCADE ON UPDATE CASCADE,
  organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  created_at timestamp without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (entry_key, organization_id)
);

CREATE INDEX idx_intake_catalog_organizations_organization_id
  ON intake_catalog_organizations (organization_id);

COMMIT;
