BEGIN;

CREATE TABLE agent_chat_runs (
    id uuid PRIMARY KEY,
    chat_id uuid NOT NULL REFERENCES agent_chats(id) ON DELETE CASCADE,
    model text NOT NULL DEFAULT '',
    input_tokens bigint NOT NULL DEFAULT 0,
    output_tokens bigint NOT NULL DEFAULT 0,
    cache_read_tokens bigint NOT NULL DEFAULT 0,
    cache_write_tokens bigint NOT NULL DEFAULT 0,
    total_tokens bigint NOT NULL DEFAULT 0,
    estimated_cost_usd double precision,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_agent_chat_runs_chat_id ON agent_chat_runs (chat_id);

ALTER TABLE agent_chat_messages
    ADD COLUMN run_id uuid REFERENCES agent_chat_runs(id) ON DELETE SET NULL;

ALTER TABLE agent_chats
    ADD COLUMN total_input_tokens bigint NOT NULL DEFAULT 0,
    ADD COLUMN total_output_tokens bigint NOT NULL DEFAULT 0,
    ADD COLUMN total_tokens bigint NOT NULL DEFAULT 0,
    ADD COLUMN total_estimated_cost_usd double precision NOT NULL DEFAULT 0;

COMMIT;
