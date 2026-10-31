CREATE UNIQUE INDEX IF NOT EXISTS runners_fleet_creation_idempotency_key
    ON runners (fleet_id, creation_idempotency_key)
    WHERE creation_idempotency_key IS NOT NULL;

DROP INDEX IF EXISTS runners_creation_idempotency_key;
