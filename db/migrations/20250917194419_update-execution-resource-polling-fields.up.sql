BEGIN;

ALTER TABLE execution_resources RENAME COLUMN last_retry_at TO last_polled_at;
ALTER TABLE execution_resources DROP COLUMN retry_count;

COMMIT;