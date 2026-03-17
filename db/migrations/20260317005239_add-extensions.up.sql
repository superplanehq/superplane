BEGIN;

CREATE TABLE extensions (
  id uuid NOT NULL DEFAULT uuid_generate_v4(),
  organization_id uuid NOT NULL,
  name character varying(255) NOT NULL,
  description text NOT NULL,
  created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
  updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
  deleted_at timestamp without time zone,

  PRIMARY KEY (id),
  FOREIGN KEY (organization_id) REFERENCES organizations(id),
  UNIQUE (organization_id, name)
);

CREATE INDEX idx_extensions_organization_id ON extensions(organization_id);
CREATE INDEX idx_extensions_deleted_at ON extensions(deleted_at);

CREATE TABLE extension_versions (
  id uuid NOT NULL DEFAULT uuid_generate_v4(),
  organization_id uuid NOT NULL,
  extension_id uuid NOT NULL,
  name character varying(128) NOT NULL,
  digest character varying(128) NOT NULL,
  state character varying(64) NOT NULL,
  created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
  updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
  published_at timestamp without time zone,
  deleted_at timestamp without time zone,
  manifest jsonb NOT NULL,

  PRIMARY KEY (id),
  FOREIGN KEY (organization_id) REFERENCES organizations(id),
  FOREIGN KEY (extension_id) REFERENCES extensions(id),
  UNIQUE (extension_id, name)
);

CREATE INDEX idx_extension_versions_org_extension ON extension_versions(organization_id, extension_id);

COMMIT;