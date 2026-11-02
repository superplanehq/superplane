ALTER TABLE factory_llm_model_allowlists
  ADD COLUMN inherit_parent boolean,
  DROP CONSTRAINT factory_llm_model_allowlists_funding,
  ADD CONSTRAINT factory_llm_model_allowlists_funding
    CHECK (funding_source IN ('hosted', 'byok', 'subscription')),
  ADD CONSTRAINT factory_llm_model_allowlists_subscription_provider
    CHECK (funding_source <> 'subscription' OR provider = 'openai');
