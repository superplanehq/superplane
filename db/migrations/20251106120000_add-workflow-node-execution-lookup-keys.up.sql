CREATE TABLE IF NOT EXISTS workflow_node_execution_lookup_keys (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  execution_id UUID NOT NULL REFERENCES workflow_node_executions(id) ON DELETE CASCADE,

  key TEXT NOT NULL,
  value TEXT NOT NULL,

  created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_workflow_node_execution_lookup_keys_ekv 
  ON workflow_node_execution_lookup_keys (execution_id, key, value);
