begin;

ALTER TABLE workflows
  ADD COLUMN IF NOT EXISTS is_template BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_workflows_is_template ON workflows(is_template);

commit;
