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
  draft_title             TEXT NOT NULL DEFAULT '',
  draft_description       TEXT NOT NULL DEFAULT '',
  draft_work_order_id     UUID REFERENCES factory_work_orders(id) ON DELETE SET NULL,
  wait_state              TEXT NOT NULL DEFAULT '',
  wait_kind               TEXT NOT NULL DEFAULT '',
  wait_text               TEXT NOT NULL DEFAULT '',
  wait_work_order_id      UUID,
  wait_work_order_key     TEXT NOT NULL DEFAULT '',
  survey_id               UUID,
  survey                  JSONB,
  heartbeat_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  ended_at                TIMESTAMPTZ,
  created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE factory_planning_session_messages (
  id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  session_id  UUID NOT NULL REFERENCES factory_planning_sessions(id) ON DELETE CASCADE,
  role        TEXT NOT NULL,
  text        TEXT NOT NULL,
  delivered   BOOLEAN NOT NULL DEFAULT FALSE,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE factory_planning_session_work_orders (
  session_id     UUID NOT NULL REFERENCES factory_planning_sessions(id) ON DELETE CASCADE,
  work_order_id  UUID NOT NULL REFERENCES factory_work_orders(id) ON DELETE RESTRICT,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (session_id, work_order_id)
);

CREATE INDEX idx_factory_planning_sessions_open_heartbeat
  ON factory_planning_sessions (heartbeat_at)
  WHERE state <> 'ended';

CREATE INDEX idx_factory_planning_sessions_factory_open
  ON factory_planning_sessions (organization_id, factory_id)
  WHERE state <> 'ended';

CREATE UNIQUE INDEX idx_factory_planning_sessions_canvas_run
  ON factory_planning_sessions (canvas_run_id)
  WHERE canvas_run_id IS NOT NULL;

CREATE INDEX idx_factory_planning_session_messages_session
  ON factory_planning_session_messages (session_id, created_at);

COMMIT;
