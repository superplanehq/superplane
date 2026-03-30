CREATE TABLE IF NOT EXISTS blobs (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  scope_type varchar(32) NOT NULL,
  canvas_id uuid NULL,
  node_id varchar(255) NULL,
  execution_id uuid NULL,
  path text NOT NULL,
  object_key text NOT NULL,
  size_bytes bigint NOT NULL,
  content_type text NULL,
  created_by_user_id uuid NULL,
  created_at timestamptz DEFAULT NOW(),
  updated_at timestamptz DEFAULT NOW(),
  CONSTRAINT blobs_scope_type_check
    CHECK (scope_type IN ('organization', 'canvas', 'node', 'execution'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_blobs_object_key ON blobs (object_key);
CREATE INDEX IF NOT EXISTS idx_blobs_org_scope_created_at ON blobs (organization_id, scope_type, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_blobs_canvas_scope ON blobs (organization_id, canvas_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_blobs_node_scope ON blobs (organization_id, canvas_id, node_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_blobs_execution_scope ON blobs (organization_id, execution_id, created_at DESC);
