BEGIN;

CREATE TABLE factory_planning_sessions (
  id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  organization_id         UUID NOT NULL,
  factory_id              UUID NOT NULL REFERENCES factories(id) ON DELETE RESTRICT,
  created_by_user_id      UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  repository              TEXT NOT NULL,
  state                   TEXT NOT NULL,
  canvas_id               UUID,
  canvas_run_id           UUID,
  messages                JSONB NOT NULL DEFAULT '[]'::jsonb,
  pending_draft           JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_work_order_ids  JSONB NOT NULL DEFAULT '[]'::jsonb,
  wait_state              TEXT NOT NULL DEFAULT '',
  wait_result             JSONB NOT NULL DEFAULT '{}'::jsonb,
  heartbeat_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  ended_at                TIMESTAMPTZ,
  created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_factory_planning_sessions_factory_created
  ON factory_planning_sessions (factory_id, created_at DESC);

CREATE INDEX idx_factory_planning_sessions_canvas_run
  ON factory_planning_sessions (canvas_run_id)
  WHERE canvas_run_id IS NOT NULL;

CREATE TABLE factory_planning_session_surveys (
  id                   UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  organization_id      UUID NOT NULL,
  factory_id           UUID NOT NULL REFERENCES factories(id) ON DELETE RESTRICT,
  session_id           UUID NOT NULL REFERENCES factory_planning_sessions(id) ON DELETE RESTRICT,
  canvas_run_id        UUID NOT NULL,
  status               TEXT NOT NULL,
  questions            JSONB NOT NULL DEFAULT '[]'::jsonb,
  answers              JSONB NOT NULL DEFAULT '[]'::jsonb,
  timeout_seconds      INTEGER NOT NULL,
  expires_at           TIMESTAMPTZ NOT NULL,
  created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  answered_at          TIMESTAMPTZ,
  answered_by_user_id  UUID REFERENCES users(id) ON DELETE RESTRICT
);

CREATE UNIQUE INDEX idx_factory_planning_session_surveys_one_pending
  ON factory_planning_session_surveys (session_id)
  WHERE status = 'pending';

CREATE INDEX idx_factory_planning_session_surveys_session_created
  ON factory_planning_session_surveys (session_id, created_at DESC);

COMMIT;
