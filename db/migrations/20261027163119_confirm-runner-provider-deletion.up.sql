ALTER TABLE runners
  ADD COLUMN provider_deleted_at TIMESTAMPTZ;

ALTER TABLE runners
  ADD CONSTRAINT runners_provider_deleted_check
  CHECK (provider_deleted_at IS NULL OR state = 'terminated');
