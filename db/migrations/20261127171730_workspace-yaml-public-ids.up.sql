ALTER TABLE workflows ADD COLUMN configuration_id text NOT NULL DEFAULT '';

-- Adopt existing records once. Allocation checks the persisted IDs so names
-- that normalize to the same text keep separate identities.
DO $$
DECLARE
  automation record;
  base text;
  candidate text;
  suffix integer;
BEGIN
  FOR automation IN
    SELECT id, factory_id, name FROM workflows
    WHERE factory_id IS NOT NULL ORDER BY factory_id, created_at, id
  LOOP
    base := COALESCE(NULLIF(btrim(left(regexp_replace(lower(automation.name), '[^a-z0-9]+', '-', 'g'), 63), '-'), ''), 'automation');
    candidate := base;
    suffix := 1;
    WHILE EXISTS (SELECT 1 FROM workflows WHERE factory_id = automation.factory_id AND configuration_id = candidate) LOOP
      suffix := suffix + 1;
      candidate := base || '-' || suffix;
    END LOOP;
    UPDATE workflows SET configuration_id = candidate WHERE id = automation.id;
  END LOOP;
END $$;

ALTER TABLE workflows ADD CONSTRAINT workflows_configuration_id_format
  CHECK (configuration_id = '' OR configuration_id ~ '^[a-z0-9]+(-[a-z0-9]+)*$');
CREATE UNIQUE INDEX workflows_factory_configuration_id_active_key
  ON workflows(factory_id, configuration_id)
  WHERE factory_id IS NOT NULL AND deleted_at IS NULL AND configuration_id <> '';

DO $$
DECLARE
  line record;
  stage jsonb;
  next_stages jsonb;
  base text;
  candidate text;
  suffix integer;
BEGIN
  FOR line IN SELECT id, factory_id, stages FROM factory_lines ORDER BY factory_id, created_at, id LOOP
    next_stages := '[]'::jsonb;
    FOR stage IN SELECT value FROM jsonb_array_elements(line.stages) LOOP
      base := COALESCE(NULLIF(btrim(left(regexp_replace(lower(stage->>'name'), '[^a-z0-9]+', '-', 'g'), 63), '-'), ''), 'stage');
      candidate := base;
      suffix := 1;
      WHILE EXISTS (
        SELECT 1 FROM factory_lines other, jsonb_array_elements(other.stages) item
        WHERE other.factory_id = line.factory_id AND item->>'configuration_id' = candidate
      ) OR EXISTS (SELECT 1 FROM jsonb_array_elements(next_stages) item WHERE item->>'configuration_id' = candidate) LOOP
        suffix := suffix + 1;
        candidate := base || '-' || suffix;
      END LOOP;
      next_stages := next_stages || jsonb_build_array(jsonb_set(stage, '{configuration_id}', to_jsonb(candidate)));
    END LOOP;
    UPDATE factory_lines SET stages = next_stages WHERE id = line.id;
  END LOOP;
END $$;

-- Several older line-scoped identities can own one shared canvas. Retain its
-- binding once and use the adopted workspace-wide identity.
DELETE FROM factory_configuration_automations duplicate
USING factory_configuration_automations retained
WHERE duplicate.factory_id = retained.factory_id
  AND duplicate.canvas_id = retained.canvas_id
  AND (duplicate.line_id, duplicate.stage_key, duplicate.manifest_id) >
      (retained.line_id, retained.stage_key, retained.manifest_id);
ALTER TABLE factory_configuration_automations DROP CONSTRAINT factory_configuration_automations_pkey;
UPDATE factory_configuration_automations ownership
SET manifest_id = automation.configuration_id
FROM workflows automation WHERE automation.id = ownership.canvas_id;
ALTER TABLE factory_configuration_automations ADD PRIMARY KEY (factory_id, manifest_id);
CREATE UNIQUE INDEX factory_configuration_automations_canvas_key
  ON factory_configuration_automations(factory_id, canvas_id);

ALTER TABLE factories DROP CONSTRAINT factories_organization_id_name_key;
