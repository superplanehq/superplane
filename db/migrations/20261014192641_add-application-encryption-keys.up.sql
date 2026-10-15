BEGIN;

CREATE TABLE application_encryption_keys (
  id UUID PRIMARY KEY,
  wrapped_key BYTEA NOT NULL,
  master_key_ref TEXT NOT NULL,
  state TEXT NOT NULL CHECK (state IN ('active', 'available')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX application_encryption_keys_one_active
  ON application_encryption_keys ((1)) WHERE state = 'active';

CREATE TABLE application_encryption_migrations (
  id UUID PRIMARY KEY,
  target_key_id UUID NOT NULL REFERENCES application_encryption_keys(id),
  state TEXT NOT NULL CHECK (state IN ('pending', 'running', 'failed', 'completed', 'abandoned')),
  registry_version INTEGER NOT NULL,
  storage_location TEXT NOT NULL,
  cursor TEXT NOT NULL DEFAULT '',
  pass INTEGER NOT NULL DEFAULT 1,
  error_message TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  completed_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX application_encryption_migrations_unfinished
  ON application_encryption_migrations ((1))
  WHERE state IN ('pending', 'running', 'failed');

COMMIT;
