BEGIN;

ALTER TABLE agent_sessions
    ADD COLUMN tracked_usage_input_tokens BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN tracked_usage_output_tokens BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN tracked_usage_cache_read_tokens BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN tracked_usage_cache_write_tokens BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN tracked_usage_total_tokens BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN tracked_usage_initialized BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE agent_sessions
    ALTER COLUMN tracked_usage_initialized SET DEFAULT TRUE;

COMMIT;
