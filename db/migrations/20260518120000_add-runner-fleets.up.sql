BEGIN;

CREATE TABLE IF NOT EXISTS runner_fleets (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name       VARCHAR(255) NOT NULL,
    auth_token TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX runner_fleets_name_idx ON runner_fleets (name);

CREATE TABLE IF NOT EXISTS runner_tasks (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    fleet_id      UUID NOT NULL REFERENCES runner_fleets(id) ON DELETE CASCADE,
    fleet_task_id TEXT NOT NULL,
    execution_id  UUID NOT NULL REFERENCES workflow_node_executions(id) ON DELETE CASCADE,
    status        VARCHAR(32) NOT NULL DEFAULT 'queued',
    spec          JSONB NOT NULL DEFAULT '{}'::jsonb,
    exit_code     INTEGER,
    output        TEXT NOT NULL DEFAULT '',
    error         TEXT NOT NULL DEFAULT '',
    result        JSONB,
    task_log      JSONB,
    dispatched_at TIMESTAMPTZ,
    completed_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX runner_tasks_fleet_task_idx ON runner_tasks (fleet_id, fleet_task_id);
CREATE INDEX runner_tasks_execution_id_idx ON runner_tasks (execution_id);
CREATE INDEX runner_tasks_fleet_status_created_idx ON runner_tasks (fleet_id, status, created_at);

COMMIT;
