BEGIN;

CREATE TABLE runner_pools (
  id uuid NOT NULL DEFAULT uuid_generate_v4(),
  organization_id uuid NOT NULL,
  name character varying(128) NOT NULL,
  created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
  updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,

  PRIMARY KEY (id),
  FOREIGN KEY (organization_id) REFERENCES organizations(id),
  UNIQUE (organization_id, name)
);

CREATE INDEX idx_runner_pools_organization_id ON runner_pools(organization_id);

CREATE TABLE runners (
  id uuid NOT NULL DEFAULT uuid_generate_v4(),
  organization_id uuid NOT NULL,
  pool_id uuid NOT NULL,
  state character varying(64) NOT NULL,
  created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
  updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,

  PRIMARY KEY (id),
  FOREIGN KEY (organization_id) REFERENCES organizations(id),
  FOREIGN KEY (pool_id) REFERENCES runner_pools(id)
);

CREATE INDEX idx_runners_organization_id ON runners(organization_id);

CREATE TABLE runner_jobs (
  id uuid NOT NULL DEFAULT uuid_generate_v4(),
  organization_id uuid NOT NULL,
  type character varying(64) NOT NULL,
  spec jsonb NOT NULL,
  state character varying(64) NOT NULL,
  result character varying(64) NOT NULL,
  result_reason character varying(255) NOT NULL,
  runner_id uuid,
  reference_id uuid NOT NULL,
  created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
  updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,

  PRIMARY KEY (id),
  FOREIGN KEY (organization_id) REFERENCES organizations(id),
  FOREIGN KEY (runner_id) REFERENCES runners(id)
);

COMMIT;