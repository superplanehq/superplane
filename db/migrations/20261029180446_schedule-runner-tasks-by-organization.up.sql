BEGIN;
ALTER TABLE runner_tasks DROP CONSTRAINT runner_tasks_state_check;
ALTER TABLE runner_tasks ADD CONSTRAINT runner_tasks_state_check
  CHECK (state IN ('pending', 'queued', 'reserved', 'running', 'succeeded', 'failed', 'canceled', 'lost'));
ALTER TABLE runner_tasks ADD COLUMN ready_at timestamp with time zone;

-- Existing reservations retain their slots. Unassigned work must acquire a
-- slot under the new scheduler before runners can claim it.
UPDATE runner_tasks SET state = 'pending'
  WHERE backend = 'integrated' AND state = 'queued' AND runner_id IS NULL;
UPDATE runner_tasks SET ready_at = queued_at
  WHERE backend = 'integrated' AND state <> 'pending';

DROP INDEX runner_tasks_organization_active_idx;
CREATE INDEX runner_tasks_organization_active_idx ON runner_tasks (organization_id)
  WHERE backend = 'integrated' AND state IN ('queued', 'reserved', 'running');
CREATE INDEX runner_tasks_organization_pending_idx ON runner_tasks (organization_id, queued_at, id)
  WHERE backend = 'integrated' AND state = 'pending';
COMMIT;
