BEGIN;

ALTER TABLE factory_planning_sessions
  ALTER COLUMN created_by_user_id DROP NOT NULL;

ALTER TABLE factory_planning_sessions
  ADD CONSTRAINT factory_planning_sessions_task_creation_creator_check
  CHECK (
    kind <> 'task_creation'
    OR created_by_user_id IS NOT NULL
  );

COMMIT;
