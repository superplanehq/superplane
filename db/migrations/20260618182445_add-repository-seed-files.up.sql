BEGIN;

CREATE TABLE IF NOT EXISTS repository_seed_files (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  repository_id   UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
  path            TEXT NOT NULL,
  content         BYTEA NOT NULL,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  UNIQUE (repository_id, path)
);

CREATE INDEX IF NOT EXISTS idx_repository_seed_files_repository_id
  ON repository_seed_files (repository_id);

COMMIT;
