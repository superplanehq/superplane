CREATE TABLE IF NOT EXISTS workflow_node_execution_consumed_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  execution_id UUID NOT NULL REFERENCES workflow_node_executions(id) ON DELETE CASCADE,
  event_id UUID NOT NULL REFERENCES workflow_events(id) ON DELETE CASCADE,

  created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_workflow_node_execution_consumed_events_execution_id
  ON workflow_node_execution_consumed_events (execution_id);

CREATE INDEX IF NOT EXISTS idx_workflow_node_execution_consumed_events_event_id
  ON workflow_node_execution_consumed_events (event_id);
