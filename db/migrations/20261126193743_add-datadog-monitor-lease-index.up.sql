CREATE INDEX CONCURRENTLY idx_workflow_node_requests_datadog_monitor_lease
ON workflow_node_requests (workflow_id, node_id, run_at DESC, id)
WHERE spec->'datadog_monitor'->>'lease_until' IS NOT NULL;
