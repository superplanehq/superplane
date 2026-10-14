-- Checks startup and webhook delivery look up one pull request head
-- while they hold the duplicate lock. Index that identity so the
-- lookup does not scan the event history for the canvas node.
ALTER TABLE public.workflow_events
  ADD COLUMN dedup_key text;

-- Events stored before this column existed have no key. A delayed
-- webhook must still find the original checks event.
UPDATE public.workflow_events AS events
SET dedup_key =
  lower(btrim(events.data #>> '{data,repository,full_name}'))
  || '|'
  || trunc((events.data #>> '{data,pull_request,number}')::numeric)::bigint::text
  || '|'
  || btrim(events.data #>> '{data,pull_request,head,sha}')
WHERE events.dedup_key IS NULL
  AND events.data ->> 'type' = 'github.pullRequest'
  AND btrim(coalesce(events.data #>> '{data,repository,full_name}', '')) <> ''
  AND btrim(coalesce(events.data #>> '{data,pull_request,head,sha}', '')) <> ''
  AND coalesce(events.data #>> '{data,pull_request,number}', '') ~ '^[0-9]+(\.0+)?$'
  AND events.workflow_id IN (
    SELECT canvas_id
    FROM public.factory_pr_feedback_handlers
    WHERE source = 'pull-request-checks'
      AND subject = 'github-pull-request'
  );

CREATE INDEX idx_workflow_events_dedup_key
  ON public.workflow_events (workflow_id, node_id, dedup_key)
  WHERE dedup_key IS NOT NULL;

CREATE INDEX idx_workflow_events_legacy_pull_request
  ON public.workflow_events (workflow_id, node_id)
  WHERE dedup_key IS NULL
    AND (data ->> 'type') = 'github.pullRequest';
