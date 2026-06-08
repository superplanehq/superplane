begin;

ALTER TABLE workflow_node_executions
  ADD COLUMN IF NOT EXISTS guardrail_scan_id   UUID,
  ADD COLUMN IF NOT EXISTS guardrail_blocked_at TIMESTAMP WITH TIME ZONE;

commit;
