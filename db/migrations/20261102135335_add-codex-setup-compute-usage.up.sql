ALTER TABLE runner_tasks
  ADD COLUMN setup_purpose text NOT NULL DEFAULT '',
  ADD COLUMN setup_billable boolean NOT NULL DEFAULT false,
  ADD CONSTRAINT runner_tasks_setup_purpose_known
    CHECK (setup_purpose IN ('', 'codex_login', 'codex_catalog')),
  ADD CONSTRAINT runner_tasks_setup_billable_shape
    CHECK (NOT setup_billable OR setup_purpose <> '');

ALTER TABLE workspace_usage_events
  ALTER COLUMN canvas_run_id DROP NOT NULL,
  ALTER COLUMN node_execution_id DROP NOT NULL,
  ADD COLUMN runner_task_id uuid REFERENCES runner_tasks(id),
  ADD COLUMN setup_purpose text NOT NULL DEFAULT '',
  ADD CONSTRAINT workspace_usage_events_scope_shape CHECK (
    (setup_purpose = '' AND runner_task_id IS NULL
      AND canvas_run_id IS NOT NULL AND node_execution_id IS NOT NULL)
    OR
    (setup_purpose IN ('codex_login', 'codex_catalog') AND runner_task_id IS NOT NULL
      AND factory_id IS NULL AND work_order_id IS NULL AND line_id IS NULL
      AND line_dispatch_id IS NULL AND work_order_execution_id IS NULL
      AND canvas_run_id IS NULL AND node_execution_id IS NULL AND node_id = ''
      AND usage_kind = 'compute' AND provider = 'runner' AND funding_source = 'hosted'
      AND input_tokens = 0 AND output_tokens = 0 AND cache_read_tokens = 0
      AND cache_write_tokens = 0 AND reasoning_tokens = 0 AND total_tokens = 0)
  );

CREATE INDEX workspace_usage_events_runner_task_id
  ON workspace_usage_events (runner_task_id) WHERE runner_task_id IS NOT NULL;
