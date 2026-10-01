BEGIN;

CREATE TABLE organization_hosted_model_allowlists (
  organization_id UUID NOT NULL,
  provider        TEXT NOT NULL,
  allowed_models  JSONB NOT NULL DEFAULT '[]'::jsonb,
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (organization_id, provider),
  CONSTRAINT organization_hosted_model_allowlists_known_provider
    CHECK (provider IN ('anthropic', 'openai', 'openrouter')),
  CONSTRAINT organization_hosted_model_allowlists_organization_id_fkey
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE
);

COMMIT;
