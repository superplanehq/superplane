BEGIN;

ALTER TABLE app_installations ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE;
CREATE INDEX idx_app_installations_deleted_at ON app_installations (deleted_at);

COMMIT;
