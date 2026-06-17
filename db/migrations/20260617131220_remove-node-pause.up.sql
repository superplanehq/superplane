UPDATE workflow_nodes
SET
  state = CASE
    WHEN EXISTS (
      SELECT 1
      FROM workflow_node_executions
      WHERE workflow_node_executions.workflow_id = workflow_nodes.workflow_id
        AND workflow_node_executions.node_id = workflow_nodes.node_id
        AND workflow_node_executions.state = 'started'
    ) THEN 'processing'
    ELSE 'ready'
  END,
  updated_at = NOW()
WHERE state = 'paused';
