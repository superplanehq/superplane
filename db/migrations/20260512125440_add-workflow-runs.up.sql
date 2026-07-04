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
  FOREIGN KEY (workflow_id) REFERENCES workflows(id) ON DELETE CASCADE
);

ALTER TABLE workflow_events ADD COLUMN run_id uuid;
ALTER TABLE workflow_node_queue_items ADD COLUMN run_id uuid;
ALTER TABLE workflow_node_executions ADD COLUMN run_id uuid;

-- Historical retention/cascade behavior can leave runtime rows without a
-- root event. Those rows cannot be associated with a workflow run, so remove
-- them before enforcing run_id as NOT NULL.
DELETE FROM workflow_node_queue_items
WHERE root_event_id IS NULL;

DELETE FROM workflow_node_executions
WHERE root_event_id IS NULL;

CREATE TEMP TABLE workflow_run_backfill (
  root_event_id uuid PRIMARY KEY,
  run_id        uuid NOT NULL
) ON COMMIT DROP;

INSERT INTO workflow_run_backfill (root_event_id, run_id)
SELECT workflow_events.id, uuid_generate_v4()
FROM workflow_events
WHERE workflow_events.execution_id IS NULL;

INSERT INTO workflow_runs (
  id,
  workflow_id,
  state,
  result,
  created_at,
  updated_at,
  finished_at
)
SELECT
  workflow_run_backfill.run_id,
  workflow_events.workflow_id,
  'finished',
  'passed',
  workflow_events.created_at,
  workflow_events.created_at,
  workflow_events.created_at
FROM workflow_run_backfill
INNER JOIN workflow_events
  ON workflow_events.id = workflow_run_backfill.root_event_id;

UPDATE workflow_events
SET run_id = workflow_run_backfill.run_id
FROM workflow_run_backfill
WHERE workflow_events.id = workflow_run_backfill.root_event_id;

UPDATE workflow_node_executions
SET run_id = workflow_run_backfill.run_id
FROM workflow_run_backfill
WHERE workflow_node_executions.root_event_id = workflow_run_backfill.root_event_id;

UPDATE workflow_node_queue_items
SET run_id = workflow_run_backfill.run_id
FROM workflow_run_backfill
WHERE workflow_node_queue_items.root_event_id = workflow_run_backfill.root_event_id;

UPDATE workflow_events
SET run_id = workflow_node_executions.run_id
FROM workflow_node_executions
WHERE workflow_events.execution_id = workflow_node_executions.id;

ALTER TABLE workflow_events ALTER COLUMN run_id SET NOT NULL;
ALTER TABLE workflow_node_queue_items ALTER COLUMN run_id SET NOT NULL;
ALTER TABLE workflow_node_executions ALTER COLUMN run_id SET NOT NULL;

ALTER TABLE workflow_events
  ADD CONSTRAINT workflow_events_run_id_fkey
  FOREIGN KEY (run_id) REFERENCES workflow_runs(id);

ALTER TABLE workflow_node_queue_items
  ADD CONSTRAINT workflow_node_queue_items_run_id_fkey
  FOREIGN KEY (run_id) REFERENCES workflow_runs(id);

ALTER TABLE workflow_node_executions
  ADD CONSTRAINT workflow_node_executions_run_id_fkey
  FOREIGN KEY (run_id) REFERENCES workflow_runs(id);

CREATE INDEX idx_workflow_runs_workflow_created_at ON workflow_runs(workflow_id, created_at DESC);
CREATE INDEX idx_workflow_runs_workflow_state ON workflow_runs(workflow_id, state);
CREATE INDEX idx_workflow_events_run_id_state ON workflow_events(run_id, state);
CREATE INDEX idx_workflow_node_queue_items_run_id ON workflow_node_queue_items(run_id);
CREATE INDEX idx_workflow_node_executions_run_id_state ON workflow_node_executions(run_id, state);

COMMIT;
