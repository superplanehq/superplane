ALTER TABLE workflow_versions
  ADD COLUMN control jsonb DEFAULT '{}'::jsonb;
