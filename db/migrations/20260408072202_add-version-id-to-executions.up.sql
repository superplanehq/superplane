ALTER TABLE workflow_node_executions
  ADD COLUMN canvas_version_id UUID REFERENCES workflow_versions(id);
