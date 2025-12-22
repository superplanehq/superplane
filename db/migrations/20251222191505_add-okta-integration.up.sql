
CREATE TABLE IF NOT EXISTS organization_okta_configs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL UNIQUE REFERENCES organizations(id) ON DELETE CASCADE,
    saml_issuer TEXT NOT NULL,
    saml_certificate TEXT NOT NULL,
    scim_token_hash TEXT NOT NULL,
    enforce_sso BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
