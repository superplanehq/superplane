BEGIN;

ALTER TABLE agent_sessions
    ADD COLUMN agent_tool_schema_revision TEXT NOT NULL DEFAULT '',
    ADD COLUMN context_replayed_at TIMESTAMPTZ NULL;

COMMIT;
