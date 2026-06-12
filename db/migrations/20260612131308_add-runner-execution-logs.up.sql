CREATE TABLE IF NOT EXISTS workflow_node_execution_logs (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
  run_id UUID NOT NULL REFERENCES workflow_runs(id) ON DELETE CASCADE,
  node_id VARCHAR(128) NOT NULL,
  execution_id UUID NOT NULL REFERENCES workflow_node_executions(id) ON DELETE CASCADE,
  sequence BIGINT NOT NULL,
  type VARCHAR(32) NOT NULL,
  text TEXT,
  message TEXT,
  command_index INTEGER,
  status VARCHAR(32),
  duration_ms BIGINT,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
  UNIQUE (execution_id, sequence),
  FOREIGN KEY (workflow_id, node_id) REFERENCES workflow_nodes(workflow_id, node_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_workflow_node_execution_logs_execution_sequence
  ON workflow_node_execution_logs (execution_id, sequence);

CREATE INDEX IF NOT EXISTS idx_workflow_node_execution_logs_workflow_node_created
  ON workflow_node_execution_logs (workflow_id, node_id, created_at DESC);
