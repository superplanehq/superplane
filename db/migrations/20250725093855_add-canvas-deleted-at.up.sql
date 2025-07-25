BEGIN;

ALTER TABLE canvases ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE;
CREATE INDEX idx_canvases_deleted_at ON canvases (deleted_at);

COMMIT;