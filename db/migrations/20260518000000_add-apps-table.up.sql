CREATE TABLE IF NOT EXISTS apps (
  id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id         UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  display_name            TEXT NOT NULL,
  slug                    TEXT NOT NULL UNIQUE,
  description             TEXT NOT NULL DEFAULT '',
  canvas_id               UUID REFERENCES workflows(id) ON DELETE SET NULL,
  code_storage_repo_id    TEXT NOT NULL DEFAULT '',
  code_storage_remote_url TEXT NOT NULL DEFAULT '',
  default_branch          TEXT NOT NULL DEFAULT 'main',
  live_commit_sha         TEXT NOT NULL DEFAULT '',
  edit_session_branch     TEXT,
  sync_status             TEXT NOT NULL DEFAULT 'ok',
  sync_error              TEXT,
  created_by              UUID REFERENCES accounts(id) ON DELETE SET NULL,
  created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at              TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_apps_organization_id ON apps(organization_id);
CREATE INDEX IF NOT EXISTS idx_apps_canvas_id ON apps(canvas_id);
CREATE INDEX IF NOT EXISTS idx_apps_deleted_at ON apps(deleted_at);

CREATE TABLE IF NOT EXISTS app_docs (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  app_id     UUID NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
  path       TEXT NOT NULL,
  content    TEXT NOT NULL DEFAULT '',
  sha        TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(app_id, path)
);

CREATE INDEX IF NOT EXISTS idx_app_docs_app_id ON app_docs(app_id);
