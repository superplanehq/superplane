ALTER TABLE agent_session_messages
    ADD COLUMN images JSONB NOT NULL DEFAULT '[]'::jsonb;
