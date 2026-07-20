CREATE TABLE runner_broker_fleets (
  id text PRIMARY KEY,
  provisioner text NOT NULL,
  arch text NOT NULL,
  size text NOT NULL,
  created_at timestamptz NOT NULL
);

CREATE TABLE runner_broker_tasks (
  id text PRIMARY KEY,
  fleet_id text NOT NULL,
  run_mode text NOT NULL DEFAULT 'command_list',
  script_json text,
  message_chain_json text,
  command_json text NOT NULL,
  commands_json text,
  setup_commands_json text,
  environment_json text,
  webhook_url text NOT NULL,
  status text NOT NULL,
  created_at timestamptz NOT NULL,
  claimed_at timestamptz,
  lease_until timestamptz,
  runner_id text,
  execution_mode text NOT NULL DEFAULT 'host',
  docker_image text,
  execution_timeout_seconds integer,
  exit_code integer,
  output text,
  result_json text,
  error_message text,
  infra_retry_count integer NOT NULL DEFAULT 0,
  cancel_requested boolean NOT NULL DEFAULT false
);

CREATE INDEX idx_runner_broker_tasks_fleet_status_created
  ON runner_broker_tasks (fleet_id, status, created_at);

CREATE INDEX idx_runner_broker_tasks_status_lease_until
  ON runner_broker_tasks (status, lease_until)
  WHERE lease_until IS NOT NULL;
