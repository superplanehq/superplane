package store

const MigrationSQL = `
CREATE TABLE IF NOT EXISTS agent2_chat_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL,
    user_id UUID NOT NULL,
    canvas_id UUID NOT NULL,
    anthropic_session_id TEXT NOT NULL,
    initial_message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT idx_agent2_chat_sessions_unique
        UNIQUE (org_id, user_id, canvas_id, anthropic_session_id)
);

CREATE INDEX IF NOT EXISTS idx_agent2_chat_sessions_lookup
    ON agent2_chat_sessions (org_id, user_id, canvas_id, created_at DESC);
`
