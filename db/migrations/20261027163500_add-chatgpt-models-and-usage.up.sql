ALTER TABLE factory_codex_connections
  ADD COLUMN usage_limits JSONB NOT NULL DEFAULT '[]'::jsonb,
  ADD COLUMN usage_observed_at TIMESTAMPTZ,
  ADD COLUMN available_models JSONB NOT NULL DEFAULT '[]'::jsonb;
