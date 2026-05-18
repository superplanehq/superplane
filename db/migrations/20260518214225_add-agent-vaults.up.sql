CREATE TABLE agent_vaults (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    provider_vault_id VARCHAR(255) NOT NULL,
    provider_name VARCHAR(100) NOT NULL,
    credential_id VARCHAR(255),
    mcp_server_url TEXT NOT NULL,
    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_agent_vaults_user_org ON agent_vaults(user_id, organization_id);
CREATE UNIQUE INDEX idx_agent_vaults_user_org_provider ON agent_vaults(user_id, organization_id, provider_name);
