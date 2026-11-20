CREATE TABLE automations_as_code (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    factory_id uuid NOT NULL REFERENCES factories(id) ON DELETE CASCADE,
    name text NOT NULL,
    source text NOT NULL,
    revision bigint NOT NULL CHECK (revision > 0),
    artifact_id text NOT NULL,
    artifact_revision text NOT NULL,
    UNIQUE (organization_id, artifact_id)
);

CREATE INDEX automations_as_code_factory_id_idx ON automations_as_code (factory_id);
