ALTER TABLE agent_sessions ADD COLUMN api_token TEXT;
ALTER TABLE agent_sessions ADD COLUMN api_token_expires_at TIMESTAMPTZ;
