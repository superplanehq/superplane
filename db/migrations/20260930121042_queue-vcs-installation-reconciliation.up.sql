BEGIN;

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

COMMIT;
