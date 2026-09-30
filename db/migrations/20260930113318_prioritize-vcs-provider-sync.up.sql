BEGIN;

ALTER TABLE vcs_provider_repository_sync_jobs
  ADD COLUMN priority SMALLINT NOT NULL DEFAULT 0;

DROP INDEX vcs_provider_repository_sync_jobs_due_idx;

CREATE INDEX vcs_provider_repository_sync_jobs_due_idx
  ON vcs_provider_repository_sync_jobs (provider, priority DESC, run_at)
  WHERE locked_at IS NULL;

COMMIT;