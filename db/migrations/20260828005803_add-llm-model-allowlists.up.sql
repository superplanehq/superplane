BEGIN;

CREATE TABLE IF NOT EXISTS organization_byok_model_allowlists (
  organization_id UUID NOT NULL,
  provider        TEXT NOT NULL,
  allowed_models  JSONB NOT NULL DEFAULT '[]'::jsonb,
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (organization_id, provider),
  CONSTRAINT organization_byok_model_allowlists_known_provider
    CHECK (provider IN ('anthropic', 'openai', 'openrouter'))
);

CREATE TABLE IF NOT EXISTS factory_llm_model_allowlists (
  factory_id      UUID NOT NULL,
  provider        TEXT NOT NULL,
  funding_source  TEXT NOT NULL,
  allowed_models  JSONB NOT NULL DEFAULT '[]'::jsonb,
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (factory_id, provider, funding_source),
  CONSTRAINT factory_llm_model_allowlists_known_provider
    CHECK (provider IN ('anthropic', 'openai', 'openrouter')),
  CONSTRAINT factory_llm_model_allowlists_funding
    CHECK (funding_source IN ('hosted', 'byok'))
);

COMMIT;
