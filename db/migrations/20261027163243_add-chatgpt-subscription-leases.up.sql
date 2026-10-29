BEGIN;

CREATE TABLE factory_codex_connections (
  id                           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  organization_id              UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  factory_id                   UUID NOT NULL REFERENCES factories(id) ON DELETE CASCADE,
  owner_user_id                UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  status                       VARCHAR(32) NOT NULL,
  resume_status                VARCHAR(32) NOT NULL DEFAULT '',
  verification_url             TEXT NOT NULL DEFAULT '',
  verification_code            TEXT NOT NULL DEFAULT '',
  account_label                TEXT NOT NULL DEFAULT '',
  bundle_ciphertext            BYTEA,
  credential_version           BIGINT NOT NULL DEFAULT 0,
  final_checkpoint_version     BIGINT NOT NULL DEFAULT 0,
  reservation_generation       BIGINT NOT NULL DEFAULT 0,
  created_at                   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at                   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT factory_codex_connections_factory_key UNIQUE (factory_id),
  CONSTRAINT factory_codex_connections_status_check
    CHECK (status IN ('pending', 'finishing', 'ready', 'reconnect_required', 'disconnected')),
  CONSTRAINT factory_codex_connections_version_check
    CHECK (
      credential_version >= 0
      AND final_checkpoint_version >= 0
      AND final_checkpoint_version <= credential_version
      AND reservation_generation >= 0
    )
);

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

COMMIT;
