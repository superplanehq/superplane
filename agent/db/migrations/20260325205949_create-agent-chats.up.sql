BEGIN;

CREATE TABLE agent_chats (
    id TEXT PRIMARY KEY,
    org_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    canvas_id TEXT NOT NULL,
    initial_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_agent_chats_owner_canvas_created ON agent_chats (org_id, user_id, canvas_id, created_at DESC);

CREATE TABLE agent_chat_messages (
    id TEXT PRIMARY KEY,
    chat_id TEXT NOT NULL REFERENCES agent_chats(id) ON DELETE CASCADE,
    message JSONB NOT NULL,
    sort_index BIGINT NOT NULL CHECK (sort_index > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (chat_id, sort_index)
);

CREATE INDEX idx_agent_chat_messages_chat_sort ON agent_chat_messages (chat_id, sort_index);

COMMIT;