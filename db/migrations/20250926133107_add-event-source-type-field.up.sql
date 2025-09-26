BEGIN;

-- Add the type column with default value
ALTER TABLE event_sources ADD COLUMN type CHARACTER VARYING(64) NOT NULL DEFAULT 'webhook';

-- Event sources with a schedule are of type 'scheduled'
UPDATE event_sources SET type = 'scheduled' WHERE schedule IS NOT NULL;

-- Event sources with a resource_id are of type 'integration resources
UPDATE event_sources SET type = 'integration-resource' WHERE resource_id IS NOT NULL;

COMMIT;