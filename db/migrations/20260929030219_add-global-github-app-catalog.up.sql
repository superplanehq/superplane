BEGIN;

CREATE TABLE github_app_installations (
  installation_id BIGINT PRIMARY KEY,
  account_id BIGINT,
  account_login TEXT NOT NULL DEFAULT '',
  account_type TEXT NOT NULL DEFAULT '',
  html_url TEXT NOT NULL DEFAULT '',
  repository_selection TEXT NOT NULL DEFAULT '',
  suspended_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX github_app_installations_account_idx
  ON github_app_installations (account_id)
  WHERE account_id IS NOT NULL;

CREATE TABLE github_app_repositories (
  repository_id BIGINT PRIMARY KEY,
  installation_id BIGINT NOT NULL REFERENCES github_app_installations(installation_id) ON DELETE CASCADE,
  full_name TEXT NOT NULL,
  private BOOLEAN NOT NULL DEFAULT FALSE,
  default_branch TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT github_app_repositories_full_name_present CHECK (BTRIM(full_name) <> '')
);

CREATE INDEX github_app_repositories_installation_idx
  ON github_app_repositories (installation_id);

CREATE UNIQUE INDEX github_app_repositories_full_name_idx
  ON github_app_repositories (LOWER(full_name));

CREATE TABLE github_app_repository_collaborators (
  repository_id BIGINT NOT NULL REFERENCES github_app_repositories(repository_id) ON DELETE CASCADE,
  github_user_id BIGINT NOT NULL,
  github_login TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (repository_id, github_user_id)
);

CREATE INDEX github_app_repository_collaborators_user_idx
  ON github_app_repository_collaborators (github_user_id);

CREATE TABLE github_app_install_requests (
  request_id BIGINT PRIMARY KEY,
  account_id BIGINT,
  account_login TEXT NOT NULL DEFAULT '',
  account_type TEXT NOT NULL DEFAULT '',
  requester_id BIGINT NOT NULL,
  requester_login TEXT NOT NULL DEFAULT '',
  requested_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX github_app_install_requests_requester_idx
  ON github_app_install_requests (requester_id);

CREATE TABLE github_app_repository_sync_jobs (
  repository_id BIGINT PRIMARY KEY REFERENCES github_app_repositories(repository_id) ON DELETE CASCADE,
  run_at TIMESTAMPTZ NOT NULL,
  attempts INTEGER NOT NULL DEFAULT 0,
  locked_at TIMESTAMPTZ,
  last_error TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX github_app_repository_sync_jobs_due_idx
  ON github_app_repository_sync_jobs (run_at)
  WHERE locked_at IS NULL;

CREATE TABLE github_app_reconcile_jobs (
  id SMALLINT PRIMARY KEY CHECK (id = 1),
  run_at TIMESTAMPTZ NOT NULL,
  attempts INTEGER NOT NULL DEFAULT 0,
  locked_at TIMESTAMPTZ,
  last_error TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE github_app_integration_bindings (
  integration_id UUID PRIMARY KEY REFERENCES app_installations(id) ON DELETE CASCADE,
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  installation_id BIGINT NOT NULL REFERENCES github_app_installations(installation_id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX github_app_integration_bindings_installation_idx
  ON github_app_integration_bindings (installation_id);

CREATE INDEX github_app_integration_bindings_organization_installation_idx
  ON github_app_integration_bindings (organization_id, installation_id);

INSERT INTO github_app_installations (installation_id, account_login, created_at, updated_at)
SELECT DISTINCT
  (metadata->>'installationId')::BIGINT,
  COALESCE(metadata->>'owner', ''),
  COALESCE(created_at, NOW()),
  COALESCE(updated_at, NOW())
FROM app_installations
WHERE app_name = 'github'
  AND state = 'ready'
  AND COALESCE((metadata->>'hostedApp')::BOOLEAN, FALSE)
  AND metadata->>'installationId' ~ '^[0-9]+$'
ON CONFLICT (installation_id) DO NOTHING;

INSERT INTO github_app_integration_bindings (integration_id, organization_id, installation_id, created_at, updated_at)
SELECT
  id,
  organization_id,
  (metadata->>'installationId')::BIGINT,
  COALESCE(created_at, NOW()),
  COALESCE(updated_at, NOW())
FROM app_installations
WHERE app_name = 'github'
  AND state = 'ready'
  AND COALESCE((metadata->>'hostedApp')::BOOLEAN, FALSE)
  AND metadata->>'installationId' ~ '^[0-9]+$';

DELETE FROM app_installations
WHERE app_name = 'github'
  AND COALESCE((metadata->>'hostedApp')::BOOLEAN, FALSE)
  AND (
    state <> 'ready'
    OR metadata->>'installationId' IS NULL
    OR metadata->>'installationId' !~ '^[0-9]+$'
  );

UPDATE app_installations
SET metadata = '{"hostedApp": true}'::JSONB,
    updated_at = NOW()
WHERE app_name = 'github'
  AND state = 'ready'
  AND COALESCE((metadata->>'hostedApp')::BOOLEAN, FALSE)
  AND metadata->>'installationId' ~ '^[0-9]+$';

COMMIT;
