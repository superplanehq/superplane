-- superseded by 20251107100000 rename migration; kept for history
CREATE TABLE IF NOT EXISTS workflow_node_execution_kvs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  execution_id UUID NOT NULL REFERENCES workflow_node_executions(id) ON DELETE CASCADE,

  key TEXT NOT NULL,
  value TEXT NOT NULL,

  created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_workflow_node_execution_kvs_ekv 
  ON workflow_node_execution_kvs (execution_id, key, value);
