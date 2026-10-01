BEGIN;

ALTER TABLE organization_byok_model_allowlists
  DROP CONSTRAINT organization_byok_model_allowlists_known_provider;

ALTER TABLE organization_byok_model_allowlists
  ADD CONSTRAINT organization_byok_model_allowlists_known_provider
    CHECK (provider IN ('anthropic', 'openai', 'openrouter', 'custom'));

ALTER TABLE factory_llm_model_allowlists
  DROP CONSTRAINT factory_llm_model_allowlists_known_provider;

ALTER TABLE factory_llm_model_allowlists
  ADD CONSTRAINT factory_llm_model_allowlists_known_provider
    CHECK (provider IN ('anthropic', 'openai', 'openrouter', 'custom'));

COMMIT;
