BEGIN;

CREATE TABLE IF NOT EXISTS fleets (
    id TEXT PRIMARY KEY,
    provisioner TEXT NOT NULL,
    arch TEXT NOT NULL,
    size TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    fleet_id TEXT NOT NULL,
    run_mode TEXT NOT NULL DEFAULT 'command_list',
    script_json TEXT,
    message_chain_json TEXT,
    command_json TEXT NOT NULL,
    commands_json TEXT,
    setup_commands_json TEXT,
    environment_json TEXT,
    files_json TEXT,
    webhook_url TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    claimed_at TIMESTAMPTZ,
    lease_until TIMESTAMPTZ,
    runner_id TEXT,
    execution_mode TEXT NOT NULL DEFAULT 'host',
    docker_image TEXT,
    execution_timeout_seconds INTEGER,
    exit_code INTEGER,
    output TEXT,
    result_json TEXT,
    error_message TEXT,
    infra_retry_count INTEGER NOT NULL DEFAULT 0,
    cancel_requested BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS idx_tasks_fleet_status_created
    ON tasks (fleet_id, status, created_at);

COMMIT;
