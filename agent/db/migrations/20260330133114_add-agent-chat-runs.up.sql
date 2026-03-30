BEGIN;

CREATE TABLE agent_chat_runs (
    id uuid PRIMARY KEY,
    chat_id uuid NOT NULL REFERENCES agent_chats(id) ON DELETE CASCADE,
    turn_index INTEGER NOT NULL,
    model TEXT,
    provider_name TEXT,
    usage JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_agent_chat_runs_chat_id_turn_index ON agent_chat_runs (chat_id, turn_index);
CREATE UNIQUE INDEX idx_agent_chat_runs_id_chat_id ON agent_chat_runs (id, chat_id);

ALTER TABLE agent_chat_messages
    ADD COLUMN run_id uuid,
    ADD COLUMN request_usage JSONB;

CREATE INDEX idx_agent_chat_messages_run_id ON agent_chat_messages (run_id);

ALTER TABLE agent_chat_messages
    ADD CONSTRAINT agent_chat_messages_run_id_chat_id_fkey
    FOREIGN KEY (run_id, chat_id) REFERENCES agent_chat_runs(id, chat_id) ON DELETE CASCADE;

COMMIT;
