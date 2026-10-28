CREATE TABLE vcs_provider_catalog_schedules (
  provider TEXT PRIMARY KEY,
  next_reconcile_at TIMESTAMPTZ NOT NULL,
  next_sweep_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
