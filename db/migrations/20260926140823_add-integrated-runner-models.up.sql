BEGIN;

CREATE TABLE runner_fleets (
  id                            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  scope_type                    VARCHAR(32) NOT NULL,
  scope_id                      UUID REFERENCES organizations(id) ON DELETE RESTRICT,
  slug                          TEXT NOT NULL,
  enabled                       BOOLEAN NOT NULL DEFAULT TRUE,
  spec                          JSONB NOT NULL DEFAULT '{}'::jsonb,
  runner_version                TEXT NOT NULL,
  created_at                    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at                    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at                    TIMESTAMPTZ,

  CONSTRAINT runner_fleets_scope_type_check
    CHECK (scope_type IN ('installation', 'organization')),
  CONSTRAINT runner_fleets_scope_check
    CHECK (
      (scope_type = 'installation' AND scope_id IS NULL)
      OR (scope_type = 'organization' AND scope_id IS NOT NULL)
    ),
  CONSTRAINT runner_fleets_spec_check
    CHECK (jsonb_typeof(spec) = 'object'),
  CONSTRAINT runner_fleets_runner_version_check
    CHECK (runner_version <> ''),
  CONSTRAINT runner_fleets_slug_check
    CHECK (slug <> '')
);

CREATE UNIQUE INDEX runner_fleets_scope_slug_key
  ON runner_fleets (
    scope_type,
    COALESCE(scope_id, '00000000-0000-0000-0000-000000000000'::uuid),
    slug
  )
  WHERE deleted_at IS NULL;

CREATE INDEX runner_fleets_scope_idx
  ON runner_fleets (scope_type, scope_id)
  WHERE deleted_at IS NULL;

CREATE TABLE runners (
  id                       UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  fleet_id                 UUID NOT NULL REFERENCES runner_fleets(id) ON DELETE RESTRICT,
  state                    VARCHAR(32) NOT NULL DEFAULT 'pending',
  runner_version           TEXT NOT NULL,
  ephemeral                BOOLEAN NOT NULL DEFAULT FALSE,
  creation_idempotency_key  TEXT,
  creation_request_hash     TEXT,
  registered_at            TIMESTAMPTZ,
  last_seen_at              TIMESTAMPTZ,
  current_connection_id    UUID,
  termination_reason       TEXT,
  created_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  terminated_at             TIMESTAMPTZ,

  CONSTRAINT runners_state_check
    CHECK (state IN ('pending', 'idle', 'busy', 'terminated')),
  CONSTRAINT runners_termination_check
    CHECK (
      (state = 'terminated' AND terminated_at IS NOT NULL)
      OR (state <> 'terminated' AND terminated_at IS NULL)
    ),
  CONSTRAINT runners_creation_idempotency_check
    CHECK (
      (creation_idempotency_key IS NULL AND creation_request_hash IS NULL)
      OR (creation_idempotency_key IS NOT NULL AND creation_request_hash IS NOT NULL)
    )
);

CREATE INDEX runners_fleet_state_idx ON runners (fleet_id, state);
CREATE INDEX runners_last_seen_idx ON runners (last_seen_at)
  WHERE state IN ('idle', 'busy');
CREATE UNIQUE INDEX runners_creation_idempotency_key
  ON runners (creation_idempotency_key)
  WHERE creation_idempotency_key IS NOT NULL;

CREATE TABLE runner_tasks (
  id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  organization_id     UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
  fleet_id            UUID NOT NULL REFERENCES runner_fleets(id) ON DELETE RESTRICT,
  runner_id           UUID REFERENCES runners(id) ON DELETE RESTRICT,
  backend             VARCHAR(32) NOT NULL,
  state               VARCHAR(32) NOT NULL DEFAULT 'queued',
  payload_ciphertext  BYTEA NOT NULL,
  result              JSONB,
  exit_code           INTEGER,
  error_message       TEXT,
  completion_hash     TEXT,
  cancel_requested_at TIMESTAMPTZ,
  queued_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  reserved_at         TIMESTAMPTZ,
  started_at          TIMESTAMPTZ,
  finished_at         TIMESTAMPTZ,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT runner_tasks_backend_check
    CHECK (backend IN ('legacy', 'integrated')),
  CONSTRAINT runner_tasks_state_check
    CHECK (state IN ('queued', 'reserved', 'running', 'succeeded', 'failed', 'canceled', 'lost'))
);

CREATE UNIQUE INDEX runner_tasks_runner_id_key
  ON runner_tasks (runner_id)
  WHERE runner_id IS NOT NULL;
CREATE INDEX runner_tasks_fleet_state_idx ON runner_tasks (fleet_id, state, queued_at);
CREATE INDEX runner_tasks_organization_idx ON runner_tasks (organization_id, created_at);

CREATE TABLE runner_credentials (
  runner_id         UUID PRIMARY KEY REFERENCES runners(id) ON DELETE CASCADE,
  access_token_hash TEXT NOT NULL UNIQUE,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  revoked_at        TIMESTAMPTZ
);

CREATE TABLE runner_registrations (
  jti         UUID PRIMARY KEY,
  runner_id   UUID NOT NULL REFERENCES runners(id) ON DELETE CASCADE,
  expires_at  TIMESTAMPTZ NOT NULL,
  consumed_at TIMESTAMPTZ,
  revoked_at  TIMESTAMPTZ,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX runner_registrations_active_runner_key
  ON runner_registrations (runner_id)
  WHERE consumed_at IS NULL AND revoked_at IS NULL;
CREATE INDEX runner_registrations_expiry_idx
  ON runner_registrations (expires_at)
  WHERE consumed_at IS NULL AND revoked_at IS NULL;

CREATE TABLE runner_task_log_lifecycles (
  task_id          UUID PRIMARY KEY REFERENCES runner_tasks(id) ON DELETE CASCADE,
  active_store     TEXT NOT NULL,
  state            VARCHAR(32) NOT NULL DEFAULT 'active',
  final_object_key TEXT,
  final_cursor     TEXT,
  truncated        BOOLEAN NOT NULL DEFAULT FALSE,
  cleanup_after    TIMESTAMPTZ,
  processing_until TIMESTAMPTZ,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT runner_task_log_lifecycles_state_check
    CHECK (state IN ('active', 'archivable', 'archiving', 'archived')),
  CONSTRAINT runner_task_log_lifecycles_active_store_check
    CHECK (active_store <> '')
);

CREATE INDEX runner_task_log_lifecycles_archiving_idx
  ON runner_task_log_lifecycles (state, updated_at)
  WHERE state IN ('archivable', 'archiving', 'archived');

COMMIT;
