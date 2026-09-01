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

COMMIT;
