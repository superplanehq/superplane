-- Checks startup and webhook delivery look up one pull request head
-- while they hold the duplicate lock. Index that identity so the
-- lookup does not scan the event history for the canvas node.
ALTER TABLE public.workflow_events
  ADD COLUMN dedup_key text;

CREATE INDEX idx_workflow_events_dedup_key
  ON public.workflow_events (workflow_id, node_id, dedup_key)
  WHERE dedup_key IS NOT NULL;
