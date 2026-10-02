BEGIN;

CREATE TABLE vcs_provider_install_request_refreshes (
  provider TEXT PRIMARY KEY,
  refresh_until TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMIT;
