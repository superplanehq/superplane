BEGIN;

ALTER TABLE runner_fleets
    ALTER COLUMN fleet_url DROP NOT NULL;

ALTER TABLE runner_fleets
    ADD COLUMN IF NOT EXISTS mode VARCHAR(32) NOT NULL DEFAULT 'bridge';

ALTER TABLE runner_tasks
    ADD COLUMN IF NOT EXISTS status VARCHAR(32) NOT NULL DEFAULT 'queued';

ALTER TABLE runner_tasks
    ADD COLUMN IF NOT EXISTS spec JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE runner_tasks
    ADD COLUMN IF NOT EXISTS exit_code INTEGER;

ALTER TABLE runner_tasks
    ADD COLUMN IF NOT EXISTS output TEXT NOT NULL DEFAULT '';

ALTER TABLE runner_tasks
    ADD COLUMN IF NOT EXISTS error TEXT NOT NULL DEFAULT '';

ALTER TABLE runner_tasks
    ADD COLUMN IF NOT EXISTS result JSONB;

ALTER TABLE runner_tasks
    ADD COLUMN IF NOT EXISTS task_log JSONB;

ALTER TABLE runner_tasks
    ADD COLUMN IF NOT EXISTS dispatched_at TIMESTAMPTZ;

ALTER TABLE runner_tasks
    ADD COLUMN IF NOT EXISTS completed_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS runner_tasks_fleet_status_created_idx
    ON runner_tasks (fleet_id, status, created_at);

COMMIT;
