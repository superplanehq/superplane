BEGIN;

ALTER TABLE vcs_provider_installation_reconcile_requesters
  DROP CONSTRAINT vcs_provider_installation_reconcile_requesters_job_fkey;

ALTER TABLE vcs_provider_installation_reconcile_requesters
  ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

COMMIT;
