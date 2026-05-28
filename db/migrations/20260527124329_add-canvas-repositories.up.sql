BEGIN;

CREATE TABLE IF NOT EXISTS canvas_repositories (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  canvas_id       UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  provider        TEXT NOT NULL,
  repo_id         TEXT NOT NULL,
  status          VARCHAR(40) NOT NULL DEFAULT 'pending',
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  UNIQUE (canvas_id),
  UNIQUE (provider, repo_id)
);

CREATE INDEX IF NOT EXISTS idx_canvas_repositories_canvas_id ON canvas_repositories (canvas_id);

COMMIT;
