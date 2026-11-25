CREATE TABLE factory_configuration_automations (
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    factory_id uuid NOT NULL REFERENCES factories(id) ON DELETE CASCADE,
    line_id uuid NOT NULL REFERENCES factory_lines(id) ON DELETE CASCADE,
    stage_key text NOT NULL,
    manifest_id text NOT NULL,
    canvas_id uuid NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    binding jsonb NOT NULL DEFAULT '{}',
    PRIMARY KEY (factory_id, line_id, stage_key, manifest_id)
);

CREATE INDEX factory_configuration_automations_canvas_id
    ON factory_configuration_automations(canvas_id);
