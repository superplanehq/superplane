BEGIN;

CREATE TABLE installation_licenses (
  id                integer NOT NULL,
  encrypted_license bytea NOT NULL,
  installed_by      uuid,
  created_at        timestamp without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at        timestamp without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT installation_licenses_pkey PRIMARY KEY (id),
  CONSTRAINT installation_licenses_singleton CHECK (id = 1),
  CONSTRAINT installation_licenses_installed_by_fkey
    FOREIGN KEY (installed_by) REFERENCES accounts(id) ON DELETE SET NULL
);

COMMIT;
