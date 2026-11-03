-- The replacement index covers GitHub and Bitbucket events.
-- Keep this concurrent operation separate from index creation.
DROP INDEX CONCURRENTLY public.idx_workflow_events_legacy_pull_request;
