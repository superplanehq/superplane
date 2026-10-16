ALTER TABLE factory_codex_connections
    ADD COLUMN usage_limits jsonb NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN usage_observed_at timestamp with time zone;
