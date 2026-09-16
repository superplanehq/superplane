BEGIN;

CREATE TABLE factory_planning_session_activities (
  id              UUID PRIMARY KEY,
  session_id      UUID NOT NULL REFERENCES factory_planning_sessions(id) ON DELETE CASCADE,
  schema_version  INTEGER NOT NULL,
  provider        TEXT NOT NULL,
  status          TEXT NOT NULL,
  last_sequence   BIGINT NOT NULL,
  snapshot        JSONB NOT NULL,
  started_at      TIMESTAMPTZ NOT NULL,
  completed_at    TIMESTAMPTZ,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_factory_planning_session_activities_session
  ON factory_planning_session_activities (session_id, started_at, id);

ALTER TABLE factory_planning_session_messages
  ADD COLUMN activity_id UUID REFERENCES factory_planning_session_activities(id) ON DELETE SET NULL;

CREATE INDEX idx_factory_planning_session_messages_activity
  ON factory_planning_session_messages (activity_id)
  WHERE activity_id IS NOT NULL;

COMMIT;
