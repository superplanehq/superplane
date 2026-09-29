BEGIN;

CREATE TABLE vcs_provider_installations (
  provider TEXT NOT NULL,
  installation_id BIGINT NOT NULL,
  account_id BIGINT,
  account_login TEXT NOT NULL DEFAULT '',
  account_type TEXT NOT NULL DEFAULT '',
  html_url TEXT NOT NULL DEFAULT '',
  repository_selection TEXT NOT NULL DEFAULT '',
  suspended_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (provider, installation_id)
);

CREATE UNIQUE INDEX vcs_provider_installations_account_idx
  ON vcs_provider_installations (provider, account_id)
  WHERE account_id IS NOT NULL;

CREATE TABLE vcs_provider_repositories (
  provider TEXT NOT NULL,
  repository_id BIGINT NOT NULL,
  installation_id BIGINT NOT NULL,
  full_name TEXT NOT NULL,
  private BOOLEAN NOT NULL DEFAULT FALSE,
  default_branch TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT vcs_provider_repositories_pkey PRIMARY KEY (provider, repository_id),
  CONSTRAINT vcs_provider_repositories_installation_fkey
    FOREIGN KEY (provider, installation_id)
    REFERENCES vcs_provider_installations(provider, installation_id)
    ON DELETE CASCADE,
  CONSTRAINT vcs_provider_repositories_full_name_present CHECK (BTRIM(full_name) <> '')
);

CREATE INDEX vcs_provider_repositories_installation_idx
  ON vcs_provider_repositories (provider, installation_id);

CREATE UNIQUE INDEX vcs_provider_repositories_full_name_idx
  ON vcs_provider_repositories (provider, LOWER(full_name));

CREATE TABLE vcs_provider_repository_collaborators (
  provider TEXT NOT NULL,
  repository_id BIGINT NOT NULL,
  provider_user_id BIGINT NOT NULL,
  provider_login TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT vcs_provider_repository_collaborators_pkey
    PRIMARY KEY (provider, repository_id, provider_user_id),
  CONSTRAINT vcs_provider_repository_collaborators_repository_fkey
    FOREIGN KEY (provider, repository_id)
    REFERENCES vcs_provider_repositories(provider, repository_id)
    ON DELETE CASCADE
);

CREATE INDEX vcs_provider_repository_collaborators_user_idx
  ON vcs_provider_repository_collaborators (provider, provider_user_id);

CREATE TABLE vcs_provider_install_requests (
  provider TEXT NOT NULL,
  request_id BIGINT NOT NULL,
  account_id BIGINT,
  account_login TEXT NOT NULL DEFAULT '',
  account_type TEXT NOT NULL DEFAULT '',
  requester_id BIGINT NOT NULL,
  requester_login TEXT NOT NULL DEFAULT '',
  requested_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (provider, request_id)
);

CREATE INDEX vcs_provider_install_requests_requester_idx
  ON vcs_provider_install_requests (provider, requester_id);

CREATE TABLE vcs_provider_repository_sync_jobs (
  provider TEXT NOT NULL,
  repository_id BIGINT NOT NULL,
  run_at TIMESTAMPTZ NOT NULL,
  attempts INTEGER NOT NULL DEFAULT 0,
  locked_at TIMESTAMPTZ,
  last_error TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT vcs_provider_repository_sync_jobs_pkey PRIMARY KEY (provider, repository_id),
  CONSTRAINT vcs_provider_repository_sync_jobs_repository_fkey
    FOREIGN KEY (provider, repository_id)
    REFERENCES vcs_provider_repositories(provider, repository_id)
    ON DELETE CASCADE
);

CREATE INDEX vcs_provider_repository_sync_jobs_due_idx
  ON vcs_provider_repository_sync_jobs (provider, run_at)
  WHERE locked_at IS NULL;

CREATE TABLE vcs_provider_reconcile_jobs (
  provider TEXT PRIMARY KEY,
  run_at TIMESTAMPTZ NOT NULL,
  attempts INTEGER NOT NULL DEFAULT 0,
  locked_at TIMESTAMPTZ,
  last_error TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE vcs_provider_integration_bindings (
  integration_id UUID PRIMARY KEY REFERENCES app_installations(id) ON DELETE CASCADE,
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  provider TEXT NOT NULL,
  installation_id BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT vcs_provider_integration_bindings_installation_fkey
    FOREIGN KEY (provider, installation_id)
    REFERENCES vcs_provider_installations(provider, installation_id)
    ON DELETE CASCADE
);

CREATE INDEX vcs_provider_integration_bindings_installation_idx
  ON vcs_provider_integration_bindings (provider, installation_id);

CREATE INDEX vcs_provider_bindings_organization_installation_idx
  ON vcs_provider_integration_bindings (organization_id, provider, installation_id);

CREATE TABLE vcs_provider_integration_repositories (
  integration_id UUID NOT NULL,
  provider TEXT NOT NULL,
  repository_id BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT vcs_provider_integration_repositories_pkey
    PRIMARY KEY (integration_id, provider, repository_id),
  CONSTRAINT vcs_provider_integration_repositories_binding_fkey
    FOREIGN KEY (integration_id)
    REFERENCES vcs_provider_integration_bindings(integration_id)
    ON DELETE CASCADE,
  CONSTRAINT vcs_provider_integration_repositories_repository_fkey
    FOREIGN KEY (provider, repository_id)
    REFERENCES vcs_provider_repositories(provider, repository_id)
    ON DELETE CASCADE
);

CREATE INDEX vcs_provider_integration_repositories_repository_idx
  ON vcs_provider_integration_repositories (provider, repository_id);

INSERT INTO vcs_provider_installations (
  provider,
  installation_id,
  account_login,
  created_at,
  updated_at
)
SELECT DISTINCT
  'github',
  (metadata->>'installationId')::BIGINT,
  COALESCE(metadata->>'owner', ''),
  COALESCE(created_at, NOW()),
  COALESCE(updated_at, NOW())
FROM app_installations
WHERE app_name = 'github'
  AND state = 'ready'
  AND COALESCE((metadata->>'hostedApp')::BOOLEAN, FALSE)
  AND metadata->>'installationId' ~ '^[0-9]+$'
ON CONFLICT (provider, installation_id) DO NOTHING;

INSERT INTO vcs_provider_repositories (
  provider,
  repository_id,
  installation_id,
  full_name,
  created_at,
  updated_at
)
SELECT DISTINCT ON ((repository->>'id')::BIGINT)
  'github',
  (repository->>'id')::BIGINT,
  (integration.metadata->>'installationId')::BIGINT,
  CASE
    WHEN BTRIM(COALESCE(repository->>'url', '')) <> '' THEN
      REGEXP_REPLACE(
        REGEXP_REPLACE(BTRIM(repository->>'url'), '^https://github.com/', '', 'i'),
        '/+$',
        ''
      )
    ELSE BTRIM(integration.metadata->>'owner') || '/' || BTRIM(repository->>'name')
  END,
  COALESCE(integration.created_at, NOW()),
  COALESCE(integration.updated_at, NOW())
FROM app_installations AS integration
CROSS JOIN LATERAL jsonb_array_elements(
  CASE
    WHEN jsonb_typeof(integration.metadata->'repositories') = 'array' THEN integration.metadata->'repositories'
    ELSE '[]'::JSONB
  END
) AS repository
WHERE integration.app_name = 'github'
  AND integration.state = 'ready'
  AND COALESCE((integration.metadata->>'hostedApp')::BOOLEAN, FALSE)
  AND integration.metadata->>'installationId' ~ '^[0-9]+$'
  AND jsonb_typeof(integration.metadata->'repositories') = 'array'
  AND repository->>'id' ~ '^[0-9]+$'
  AND (
    BTRIM(COALESCE(repository->>'url', '')) <> ''
    OR (
      BTRIM(COALESCE(integration.metadata->>'owner', '')) <> ''
      AND BTRIM(COALESCE(repository->>'name', '')) <> ''
    )
  )
ORDER BY (repository->>'id')::BIGINT, integration.updated_at DESC NULLS LAST
ON CONFLICT (provider, repository_id) DO UPDATE SET
  installation_id = EXCLUDED.installation_id,
  full_name = EXCLUDED.full_name,
  updated_at = EXCLUDED.updated_at;

INSERT INTO vcs_provider_integration_bindings (
  integration_id,
  organization_id,
  provider,
  installation_id,
  created_at,
  updated_at
)
SELECT
  id,
  organization_id,
  'github',
  (metadata->>'installationId')::BIGINT,
  COALESCE(created_at, NOW()),
  COALESCE(updated_at, NOW())
FROM app_installations
WHERE app_name = 'github'
  AND state = 'ready'
  AND COALESCE((metadata->>'hostedApp')::BOOLEAN, FALSE)
  AND metadata->>'installationId' ~ '^[0-9]+$';

WITH factory_repositories AS (
  SELECT
    factory.id AS factory_id,
    app_repository.repository_id AS app_repository_id,
    backlog_repository.repository_id AS backlog_repository_id
  FROM factories AS factory
  INNER JOIN vcs_provider_integration_bindings AS binding
    ON binding.integration_id::TEXT = factory.onboarding_config->>'vcs_integration_id'
    AND binding.provider = 'github'
  LEFT JOIN vcs_provider_repositories AS app_repository
    ON app_repository.provider = binding.provider
    AND app_repository.installation_id = binding.installation_id
    AND LOWER(app_repository.full_name) = LOWER(BTRIM(factory.onboarding_config->>'app_repository'))
  LEFT JOIN vcs_provider_repositories AS backlog_repository
    ON backlog_repository.provider = binding.provider
    AND backlog_repository.installation_id = binding.installation_id
    AND LOWER(backlog_repository.full_name) = LOWER(BTRIM(factory.onboarding_config->>'backlog_repository'))
)
UPDATE factories AS factory
SET onboarding_config = factory.onboarding_config || JSONB_STRIP_NULLS(JSONB_BUILD_OBJECT(
  'app_repository_id', selected.app_repository_id,
  'backlog_repository_id', selected.backlog_repository_id
))
FROM factory_repositories AS selected
WHERE selected.factory_id = factory.id
  AND (
    selected.app_repository_id IS NOT NULL
    OR selected.backlog_repository_id IS NOT NULL
  );

INSERT INTO vcs_provider_integration_repositories (
  integration_id,
  provider,
  repository_id,
  created_at,
  updated_at
)
SELECT DISTINCT
  integration.id,
  'github',
  (repository->>'id')::BIGINT,
  COALESCE(integration.created_at, NOW()),
  COALESCE(integration.updated_at, NOW())
FROM app_installations AS integration
CROSS JOIN LATERAL jsonb_array_elements(
  CASE
    WHEN jsonb_typeof(integration.metadata->'repositories') = 'array' THEN integration.metadata->'repositories'
    ELSE '[]'::JSONB
  END
) AS repository
INNER JOIN vcs_provider_repositories AS catalog_repository
  ON catalog_repository.provider = 'github'
  AND catalog_repository.repository_id = (repository->>'id')::BIGINT
WHERE integration.app_name = 'github'
  AND integration.state = 'ready'
  AND COALESCE((integration.metadata->>'hostedApp')::BOOLEAN, FALSE)
  AND integration.metadata->>'installationId' ~ '^[0-9]+$'
  AND jsonb_typeof(integration.metadata->'repositories') = 'array'
  AND repository->>'id' ~ '^[0-9]+$'
ON CONFLICT (integration_id, provider, repository_id) DO NOTHING;

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

ALTER TABLE account_linked_accounts
  ADD COLUMN active BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE account_linked_accounts SET active = TRUE;

DROP INDEX idx_account_linked_accounts_account_provider;

CREATE UNIQUE INDEX idx_account_linked_accounts_account_provider_identity
  ON account_linked_accounts (account_id, provider, provider_id);

CREATE UNIQUE INDEX idx_account_linked_accounts_active_provider
  ON account_linked_accounts (account_id, provider)
  WHERE active;

COMMIT;
