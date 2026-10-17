BEGIN;

ALTER TABLE runners
  ADD COLUMN provider_deleted_at TIMESTAMPTZ;

ALTER TABLE runners
  ADD CONSTRAINT runners_provider_deleted_check
  CHECK (provider_deleted_at IS NULL OR state = 'terminated');

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
  reservation_started_version  BIGINT NOT NULL DEFAULT 0,
  reservation_task_id          UUID,
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
      AND reservation_started_version >= 0
    )
);

CREATE INDEX factory_codex_connections_reservation_idx
  ON factory_codex_connections (reservation_task_id)
  WHERE reservation_task_id IS NOT NULL;

COMMIT;
