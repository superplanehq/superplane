BEGIN;

-- Speeds batched deletes of completed workflow_node_requests by updated_at.
CREATE INDEX idx_workflow_node_requests_completed_updated_at
  ON workflow_node_requests (updated_at)
  WHERE state = 'completed';

COMMIT;
