ALTER TABLE factories
  ADD COLUMN repository_analysis JSONB NOT NULL DEFAULT '{}'::jsonb;
