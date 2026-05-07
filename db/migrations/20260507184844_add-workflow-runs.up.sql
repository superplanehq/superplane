BEGIN;

CREATE TABLE workflow_runs (
  id            uuid NOT NULL DEFAULT uuid_generate_v4(),
  workflow_id   uuid NOT NULL,
  state         CHARACTER VARYING(32) NOT NULL,
  result        CHARACTER VARYING(32),
  created_at    TIMESTAMP NOT NULL,
  updated_at    TIMESTAMP NOT NULL,
  finished_at   TIMESTAMP,

  PRIMARY KEY (id),
  FOREIGN KEY (workflow_id) REFERENCES workflows(id)
);

ALTER TABLE workflow_events ADD COLUMN run_id uuid;
ALTER TABLE workflow_node_queue_items ADD COLUMN run_id uuid;
ALTER TABLE workflow_node_executions ADD COLUMN run_id uuid;

ALTER TABLE workflow_events
  ADD CONSTRAINT workflow_events_run_id_fkey
  FOREIGN KEY (run_id) REFERENCES workflow_runs(id) ON DELETE SET NULL;

ALTER TABLE workflow_node_queue_items
  ADD CONSTRAINT workflow_node_queue_items_run_id_fkey
  FOREIGN KEY (run_id) REFERENCES workflow_runs(id) ON DELETE SET NULL;

ALTER TABLE workflow_node_executions
  ADD CONSTRAINT workflow_node_executions_run_id_fkey
  FOREIGN KEY (run_id) REFERENCES workflow_runs(id) ON DELETE SET NULL;

CREATE INDEX idx_workflow_runs_workflow_created_at ON workflow_runs(workflow_id, created_at DESC);
CREATE INDEX idx_workflow_runs_workflow_state ON workflow_runs(workflow_id, state);
CREATE INDEX idx_workflow_events_run_id_state ON workflow_events(run_id, state);
CREATE INDEX idx_workflow_node_queue_items_run_id ON workflow_node_queue_items(run_id);
CREATE INDEX idx_workflow_node_executions_run_id_state ON workflow_node_executions(run_id, state);

COMMIT;
