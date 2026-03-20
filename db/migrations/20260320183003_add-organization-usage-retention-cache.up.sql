ALTER TABLE organizations
ADD COLUMN usage_retention_window_days INTEGER NULL,
ADD COLUMN usage_limits_synced_at TIMESTAMPTZ NULL;
