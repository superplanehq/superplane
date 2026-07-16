begin;

ALTER TABLE workflow_runs ADD COLUMN parent_run_id uuid REFERENCES workflow_runs(id);
ALTER TABLE workflow_runs ADD COLUMN parent_workflow_id uuid REFERENCES workflows(id);
ALTER TABLE workflow_runs ADD COLUMN parent_execution_id uuid REFERENCES workflow_node_executions(id);
ALTER TABLE workflow_runs ADD COLUMN callbacks jsonb NOT NULL DEFAULT '[]';
ALTER TABLE workflow_runs ADD COLUMN input jsonb NOT NULL DEFAULT '{}';

--
-- TODO: do I need to turn this into a foreign key?
--
ALTER TABLE workflow_runs ADD COLUMN node_id VARCHAR(255);

commit;