BEGIN;

ALTER TABLE webhooks DROP COLUMN integration_id;
ALTER TABLE webhooks DROP COLUMN resource;

DROP TABLE IF EXISTS integrations;

COMMIT;
