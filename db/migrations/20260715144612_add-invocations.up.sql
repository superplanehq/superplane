begin;

CREATE TABLE app_invocations (
  id                  uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
  caller_app_id       uuid NOT NULL,
  caller_execution_id uuid NOT NULL,
  target_canvas_id    uuid NOT NULL,
  target_node_id      CHARACTER VARYING(128) NOT NULL,
  run_id              uuid,
  state               CHARACTER VARYING(64) NOT NULL DEFAULT 'pending',
  payload             jsonb NOT NULL,
  created_at          timestamp without time zone NOT NULL DEFAULT now(),
  updated_at          timestamp without time zone NOT NULL DEFAULT now(),

  -- TODO: not sure if DELETE CASCADE is what we want here
  FOREIGN KEY (caller_app_id) REFERENCES workflows(id) ON DELETE CASCADE,
  FOREIGN KEY (caller_execution_id) REFERENCES workflow_node_executions(id) ON DELETE CASCADE,
  FOREIGN KEY (target_canvas_id) REFERENCES workflows(id) ON DELETE CASCADE,
  FOREIGN KEY (target_canvas_id, target_node_id) REFERENCES workflow_nodes(workflow_id, node_id) ON DELETE CASCADE
);

CREATE INDEX idx_app_invocations_run_id ON app_invocations(run_id);

commit;