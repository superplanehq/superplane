BEGIN;

CREATE TABLE files (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  installation_id TEXT NOT NULL,
  scope           VARCHAR(32) NOT NULL,
  organization_id UUID REFERENCES organizations(id) ON DELETE RESTRICT,
  factory_id      UUID REFERENCES factories(id) ON DELETE RESTRICT,
  work_order_id   UUID REFERENCES factory_work_orders(id) ON DELETE RESTRICT,
  filename        TEXT NOT NULL,
  content_type    TEXT NOT NULL,
  size_bytes      BIGINT NOT NULL DEFAULT 0,
  checksum        TEXT,
  storage_key     TEXT NOT NULL,
  state           VARCHAR(32) NOT NULL DEFAULT 'pending',
  created_by_id   UUID REFERENCES users(id) ON DELETE RESTRICT,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT files_storage_key_key UNIQUE (storage_key),
  CONSTRAINT files_scope_check CHECK (scope IN ('app', 'organization', 'workspace', 'task')),
  CONSTRAINT files_state_check CHECK (state IN ('pending', 'ready', 'failed')),
  CONSTRAINT files_scope_fks_check CHECK (
    (
      scope = 'app'
      AND organization_id IS NULL
      AND factory_id IS NULL
      AND work_order_id IS NULL
    )
    OR (
      scope = 'organization'
      AND organization_id IS NOT NULL
      AND factory_id IS NULL
      AND work_order_id IS NULL
    )
    OR (
      scope = 'workspace'
      AND organization_id IS NOT NULL
      AND factory_id IS NOT NULL
      AND work_order_id IS NULL
    )
    OR (
      scope = 'task'
      AND organization_id IS NOT NULL
      AND factory_id IS NOT NULL
      AND work_order_id IS NOT NULL
    )
  )
);

CREATE INDEX idx_files_factory_id ON files (factory_id);
CREATE INDEX idx_files_work_order_id ON files (work_order_id);
CREATE INDEX idx_files_organization_id ON files (organization_id);
CREATE INDEX idx_files_stale_pending ON files (updated_at)
  WHERE state IN ('pending', 'failed');

COMMIT;
