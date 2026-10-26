CREATE TABLE factory_agent_resource_refreshes (
 resource_id uuid PRIMARY KEY REFERENCES factory_agent_resources(id) ON DELETE CASCADE,
 attempt_id uuid NOT NULL,
 expires_at timestamptz NOT NULL,
 updated_at timestamptz NOT NULL
);
