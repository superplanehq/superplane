ALTER TABLE workflows ADD COLUMN IF NOT EXISTS configuration_id text NOT NULL DEFAULT '';
CREATE UNIQUE INDEX IF NOT EXISTS workflows_factory_configuration_id_active_key
  ON workflows(factory_id, configuration_id)
  WHERE factory_id IS NOT NULL AND deleted_at IS NULL AND configuration_id <> '';

ALTER TABLE factories DROP CONSTRAINT IF EXISTS factories_organization_id_name_key;

-- Keep existing ownership records in place, including databases that ran the
-- earlier workspace-wide ownership migration during development.
ALTER TABLE factory_configuration_automations DROP CONSTRAINT factory_configuration_automations_pkey;
ALTER TABLE factory_configuration_automations ADD PRIMARY KEY (factory_id, line_id, stage_key, manifest_id);
DROP INDEX IF EXISTS factory_configuration_automations_canvas_key;
