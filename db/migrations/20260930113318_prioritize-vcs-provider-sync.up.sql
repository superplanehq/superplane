BEGIN;

ALTER TABLE vcs_provider_repository_sync_jobs
  ADD COLUMN priority SMALLINT NOT NULL DEFAULT 0;

DROP INDEX vcs_provider_repository_sync_jobs_due_idx;

CREATE INDEX vcs_provider_repository_sync_jobs_due_idx
  ON vcs_provider_repository_sync_jobs (provider, priority DESC, run_at)
  WHERE locked_at IS NULL;

CREATE TABLE vcs_provider_installation_reconcile_jobs (
  provider TEXT NOT NULL,
  installation_id BIGINT NOT NULL,
  run_at TIMESTAMPTZ NOT NULL,
  attempts INTEGER NOT NULL DEFAULT 0,
  locked_at TIMESTAMPTZ,
  last_error TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT vcs_provider_installation_reconcile_jobs_pkey
    PRIMARY KEY (provider, installation_id)
);

CREATE INDEX vcs_provider_installation_reconcile_jobs_due_idx
  ON vcs_provider_installation_reconcile_jobs (provider, run_at)
  WHERE locked_at IS NULL;

CREATE TABLE vcs_provider_installation_reconcile_requesters (
  provider TEXT NOT NULL,
  installation_id BIGINT NOT NULL,
  organization_id UUID NOT NULL,
  CONSTRAINT vcs_provider_installation_reconcile_requesters_pkey
    PRIMARY KEY (provider, installation_id, organization_id),
  CONSTRAINT vcs_provider_installation_reconcile_requesters_job_fkey
    FOREIGN KEY (provider, installation_id)
    REFERENCES vcs_provider_installation_reconcile_jobs(provider, installation_id)
    ON DELETE CASCADE
);

CREATE INDEX vcs_provider_installation_reconcile_requesters_organization_idx
  ON vcs_provider_installation_reconcile_requesters (provider, organization_id);

COMMIT;
