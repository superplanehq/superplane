BEGIN;

CREATE TABLE IF NOT EXISTS repositories (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  canvas_id       UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  provider        TEXT NOT NULL,
  repo_id         TEXT NOT NULL,
  status          VARCHAR(64) NOT NULL DEFAULT 'pending',
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  UNIQUE (canvas_id, provider, repo_id)
);

CREATE INDEX IF NOT EXISTS idx_repositories_canvas_id ON repositories (canvas_id);

COMMIT;
