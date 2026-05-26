CREATE TABLE IF NOT EXISTS canvas_dashboards (
  canvas_id  UUID PRIMARY KEY REFERENCES workflows(id) ON DELETE CASCADE,
  panels     JSONB NOT NULL DEFAULT '[]'::jsonb,
  layout     JSONB NOT NULL DEFAULT '[]'::jsonb,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
