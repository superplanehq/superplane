ALTER TABLE workflow_runs
    ADD COLUMN is_replay boolean DEFAULT false NOT NULL,
    ADD COLUMN replay_source_execution_id uuid,
    ADD COLUMN replay_payload jsonb;

ALTER TABLE workflow_runs
    ADD CONSTRAINT workflow_runs_replay_source_execution_id_fkey
    FOREIGN KEY (replay_source_execution_id)
    REFERENCES workflow_node_executions(id)
    ON DELETE SET NULL;

CREATE INDEX idx_workflow_runs_replay_source_execution_id
    ON workflow_runs (replay_source_execution_id)
    WHERE replay_source_execution_id IS NOT NULL;
