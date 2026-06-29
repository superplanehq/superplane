BEGIN;

CREATE TABLE workflow_staged_files (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  version_id UUID NOT NULL REFERENCES workflow_versions(id) ON DELETE CASCADE,
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  path TEXT NOT NULL,
  content TEXT NOT NULL DEFAULT '',
  deleted BOOLEAN NOT NULL DEFAULT false,
  updated_by UUID REFERENCES users(id) ON DELETE SET NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  UNIQUE (version_id, path)
);

CREATE INDEX idx_workflow_staged_files_version_id
  ON workflow_staged_files (version_id);

COMMIT;
