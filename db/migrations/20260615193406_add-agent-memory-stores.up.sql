BEGIN;

CREATE TABLE IF NOT EXISTS agent_memory_stores (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    canvas_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    provider VARCHAR(40) NOT NULL,
    provider_memory_store_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX agent_memory_stores_scope_provider_idx
    ON agent_memory_stores (organization_id, user_id, canvas_id, provider);

CREATE INDEX agent_memory_stores_provider_id_idx
    ON agent_memory_stores (provider, provider_memory_store_id);

COMMIT;
