BEGIN;

CREATE TABLE IF NOT EXISTS agent_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL,
    user_id UUID NOT NULL,
    canvas_id UUID NOT NULL,
    provider VARCHAR(40) NOT NULL,
    provider_session_id TEXT NOT NULL,
    status VARCHAR(40) NOT NULL DEFAULT 'idle',
    last_active_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX agent_sessions_user_canvas_idx
    ON agent_sessions (organization_id, user_id, canvas_id);

CREATE INDEX agent_sessions_provider_session_id_idx
    ON agent_sessions (provider, provider_session_id);

CREATE TABLE IF NOT EXISTS agent_session_messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID NOT NULL REFERENCES agent_sessions(id) ON DELETE CASCADE,
    provider_event_id TEXT NOT NULL DEFAULT '',
    role VARCHAR(20) NOT NULL,
    content TEXT NOT NULL DEFAULT '',
    tool_call_id TEXT NOT NULL DEFAULT '',
    tool_name TEXT NOT NULL DEFAULT '',
    tool_status VARCHAR(20) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX agent_session_messages_session_idx
    ON agent_session_messages (session_id, created_at DESC, id DESC);

CREATE UNIQUE INDEX agent_session_messages_provider_event_idx
    ON agent_session_messages (session_id, provider_event_id)
    WHERE provider_event_id <> '';

COMMIT;
