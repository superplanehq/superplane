BEGIN;

CREATE TABLE installation_license_keys (
  id         integer NOT NULL,
  document   text NOT NULL,
  version    bigint NOT NULL,
  created_at timestamp without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at timestamp without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT installation_license_keys_pkey PRIMARY KEY (id),
  CONSTRAINT installation_license_keys_singleton CHECK (id = 1),
  CONSTRAINT installation_license_keys_version_positive CHECK (version > 0)
);

COMMIT;
