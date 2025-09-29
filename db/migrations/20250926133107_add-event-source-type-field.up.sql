BEGIN;

-- Add the type column with default value
ALTER TABLE event_sources ADD COLUMN type CHARACTER VARYING(64) NOT NULL DEFAULT 'webhook';

-- Event sources with a schedule are of type 'scheduled'
UPDATE event_sources SET type = 'scheduled' WHERE schedule IS NOT NULL;

-- Event sources with resource_id use the integration type as its type
update event_sources set type = (
  select integrations.type from resources
    inner join integrations on resources.integration_id = integrations.id
    where resources.id = event_sources.resource_id
) where resource_id IS NOT NULL;

COMMIT;