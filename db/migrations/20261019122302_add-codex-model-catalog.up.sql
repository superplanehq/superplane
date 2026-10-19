ALTER TABLE factory_codex_connections
    ADD COLUMN available_models jsonb NOT NULL DEFAULT '[]'::jsonb;
