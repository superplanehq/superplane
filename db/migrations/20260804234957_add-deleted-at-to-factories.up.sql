BEGIN;

ALTER TABLE factories ADD COLUMN deleted_at TIMESTAMPTZ;
CREATE INDEX idx_factories_deleted_at ON factories (deleted_at);

COMMIT;
