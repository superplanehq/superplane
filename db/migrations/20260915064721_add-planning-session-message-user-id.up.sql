BEGIN;

ALTER TABLE factory_planning_session_messages
  ADD COLUMN user_id UUID REFERENCES users(id) ON DELETE SET NULL;

COMMIT;
