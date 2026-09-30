BEGIN;

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
