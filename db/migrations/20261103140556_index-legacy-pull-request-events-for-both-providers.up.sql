-- The legacy repair query covers both providers. Build its replacement
-- index before removing the GitHub-only index, without blocking event writes.
CREATE INDEX CONCURRENTLY idx_workflow_events_legacy_vcs_pull_request
  ON public.workflow_events (workflow_id, node_id)
  WHERE dedup_key IS NULL
    AND (data ->> 'type') IN ('github.pullRequest', 'bitbucket.pullRequest');
