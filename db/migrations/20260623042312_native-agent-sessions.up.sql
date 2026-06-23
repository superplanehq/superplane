BEGIN;

CREATE TABLE IF NOT EXISTS native_agent_sessions (
    provider_session_id TEXT PRIMARY KEY,
    history JSONB NOT NULL DEFAULT '[]'::jsonb,
    awaiting BOOLEAN NOT NULL DEFAULT FALSE,
    interrupted BOOLEAN NOT NULL DEFAULT FALSE,
    steps INTEGER NOT NULL DEFAULT 0,
    last_tool_signature TEXT NOT NULL DEFAULT '',
    repeated_tool_calls INTEGER NOT NULL DEFAULT 0,
    pending_tool_names JSONB NOT NULL DEFAULT '{}'::jsonb,
    compaction_failures INTEGER NOT NULL DEFAULT 0,
    summary TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMIT;
