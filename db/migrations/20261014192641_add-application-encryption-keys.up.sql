BEGIN;

CREATE TABLE application_encryption_keys (
  id UUID PRIMARY KEY,
  wrapped_key BYTEA NOT NULL,
  master_key_ref TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE application_encryption_state (
  id INTEGER PRIMARY KEY DEFAULT 1 CHECK (id = 1),
  active_key_id UUID NOT NULL REFERENCES application_encryption_keys(id),
  generation BIGINT NOT NULL DEFAULT 1,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE application_encryption_migrations (
  id UUID PRIMARY KEY,
  target_key_id UUID NOT NULL REFERENCES application_encryption_keys(id),
  state TEXT NOT NULL CHECK (state IN ('pending', 'running', 'failed', 'completed', 'abandoned')),
  registry_version INTEGER NOT NULL,
  storage_location TEXT NOT NULL,
  cursor TEXT NOT NULL DEFAULT '',
  pass INTEGER NOT NULL DEFAULT 1,
  lease_owner UUID,
  lease_expires_at TIMESTAMPTZ,
  lease_generation BIGINT NOT NULL DEFAULT 0,
  error_message TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  completed_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX application_encryption_migrations_unfinished
  ON application_encryption_migrations ((1))
  WHERE state IN ('pending', 'running', 'failed');

COMMIT;
