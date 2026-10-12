BEGIN;

ALTER TABLE vcs_provider_installation_reconcile_requesters
  DROP CONSTRAINT IF EXISTS vcs_provider_installation_reconcile_requesters_job_fkey;

ALTER TABLE vcs_provider_installation_reconcile_requesters
  ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE TABLE IF NOT EXISTS vcs_provider_install_request_refreshes (
  provider TEXT PRIMARY KEY,
  refresh_until TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMIT;
