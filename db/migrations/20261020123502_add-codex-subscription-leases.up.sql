BEGIN;

CREATE TABLE factory_codex_leases (
  id                 UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  connection_id      UUID NOT NULL REFERENCES factory_codex_connections(id) ON DELETE CASCADE,
  task_id            UUID NOT NULL,
  generation         BIGINT NOT NULL,
  purpose            VARCHAR(16) NOT NULL,
  secret_hash        TEXT NOT NULL DEFAULT '',
  secret_ciphertext  BYTEA,
  started_version    BIGINT NOT NULL DEFAULT 0,
  created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT factory_codex_leases_task_key UNIQUE (connection_id, task_id),
  CONSTRAINT factory_codex_leases_purpose_check CHECK (purpose IN ('login', 'run')),
  CONSTRAINT factory_codex_leases_generation_check CHECK (generation >= 1 AND started_version >= 0)
);

CREATE UNIQUE INDEX factory_codex_leases_login_idx
  ON factory_codex_leases (connection_id)
  WHERE purpose = 'login';

CREATE UNIQUE INDEX factory_codex_leases_secret_idx
  ON factory_codex_leases (secret_hash)
  WHERE secret_hash <> '';

CREATE INDEX factory_codex_leases_task_idx
  ON factory_codex_leases (task_id);

-- An in-flight reservation already has the real refresh token on its VM.
-- Keep it exclusive until that task ends.
INSERT INTO factory_codex_leases (
  connection_id, task_id, generation, purpose, started_version, created_at
)
SELECT
  id,
  reservation_task_id,
  GREATEST(reservation_generation, 1),
  'login',
  reservation_started_version,
  NOW()
FROM factory_codex_connections
WHERE reservation_task_id IS NOT NULL;

DROP INDEX IF EXISTS factory_codex_connections_reservation_idx;

ALTER TABLE factory_codex_connections
  DROP CONSTRAINT factory_codex_connections_version_check;

ALTER TABLE factory_codex_connections
  DROP COLUMN reservation_task_id,
  DROP COLUMN reservation_started_version;

ALTER TABLE factory_codex_connections
  ADD CONSTRAINT factory_codex_connections_version_check
  CHECK (
    credential_version >= 0
    AND final_checkpoint_version >= 0
    AND final_checkpoint_version <= credential_version
    AND reservation_generation >= 0
  );

COMMIT;
