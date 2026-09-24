BEGIN;

ALTER TABLE organizations
    DROP COLUMN IF EXISTS usage_synced_at,
    DROP COLUMN IF EXISTS usage_limits_synced_at;

ALTER TABLE agent_sessions
    DROP COLUMN IF EXISTS tracked_usage_input_tokens,
    DROP COLUMN IF EXISTS tracked_usage_output_tokens,
    DROP COLUMN IF EXISTS tracked_usage_cache_read_tokens,
    DROP COLUMN IF EXISTS tracked_usage_cache_write_tokens,
    DROP COLUMN IF EXISTS tracked_usage_total_tokens,
    DROP COLUMN IF EXISTS tracked_usage_initialized;

UPDATE organizations
SET usage_retention_window_days = 180;

ALTER TABLE organizations
    ALTER COLUMN usage_retention_window_days SET DEFAULT 180;

COMMIT;
