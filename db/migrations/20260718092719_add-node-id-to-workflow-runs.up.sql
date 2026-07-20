BEGIN;

ALTER TABLE workflow_runs
  ADD COLUMN node_id CHARACTER VARYING(128);

UPDATE workflow_runs AS r
SET node_id = root_events.node_id
FROM (
  SELECT DISTINCT ON (run_id)
    run_id,
    node_id
  FROM workflow_events
  WHERE execution_id IS NULL
    AND node_id IS NOT NULL
  ORDER BY run_id, created_at ASC, id ASC
) AS root_events
WHERE r.id = root_events.run_id;

ALTER TABLE workflow_runs
  ADD CONSTRAINT fk_workflow_runs_workflow_node
  FOREIGN KEY (workflow_id, node_id) REFERENCES workflow_nodes(workflow_id, node_id);

CREATE INDEX idx_workflow_runs_workflow_node_id ON workflow_runs(workflow_id, node_id);

COMMIT;
