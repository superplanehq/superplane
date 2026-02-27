CREATE TABLE IF NOT EXISTS canvas_memories (
  canvas_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
  namespace TEXT NOT NULL,
  values JSONB NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_canvas_memories_canvas_namespace
  ON canvas_memories (canvas_id, namespace);
