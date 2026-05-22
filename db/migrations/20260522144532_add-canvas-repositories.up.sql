CREATE TABLE IF NOT EXISTS canvas_repositories (
  canvas_id      UUID PRIMARY KEY REFERENCES workflows(id) ON DELETE CASCADE,
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  provider       TEXT NOT NULL,
  repo_id        TEXT NOT NULL,
  default_branch TEXT NOT NULL DEFAULT 'main',
  head_sha       TEXT,
  status         TEXT NOT NULL DEFAULT 'ready',
  created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  UNIQUE (provider, repo_id),
  CHECK (provider IN ('code_storage', 'local_git')),
  CHECK (status IN ('provisioning', 'ready', 'error'))
);

CREATE INDEX IF NOT EXISTS idx_canvas_repositories_organization_id
  ON canvas_repositories (organization_id);
