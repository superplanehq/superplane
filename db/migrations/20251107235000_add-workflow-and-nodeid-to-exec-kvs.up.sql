BEGIN;

ALTER TABLE workflow_node_execution_kvs 
  ADD COLUMN workflow_id UUID NOT NULL,
  ADD COLUMN node_id CHARACTER VARYING(128) NOT NULL;

ALTER TABLE workflow_node_execution_kvs
  ADD CONSTRAINT fk_wnek_workflow FOREIGN KEY (workflow_id) REFERENCES workflows(id) ON DELETE CASCADE,
  ADD CONSTRAINT fk_wnek_workflow_node FOREIGN KEY (workflow_id, node_id) REFERENCES workflow_nodes(workflow_id, node_id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_workflow_node_execution_kvs_workflow_node_key_value ON workflow_node_execution_kvs (workflow_id, node_id, key, value);

COMMIT;
