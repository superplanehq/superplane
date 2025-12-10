begin;

ALTER TABLE workflow_nodes
  ADD COLUMN app_installation_id uuid,
  ADD FOREIGN KEY (app_installation_id) REFERENCES app_installations(id) ON DELETE SET NULL;

CREATE INDEX idx_workflow_node_installation_id ON workflow_nodes(app_installation_id);

commit;