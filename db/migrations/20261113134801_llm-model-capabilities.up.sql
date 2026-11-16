CREATE TABLE llm_model_capabilities (
    provider text NOT NULL,
    base_url text NOT NULL,
    model text NOT NULL,
    thinking_levels jsonb NOT NULL DEFAULT '[]',
    default_thinking_level text NOT NULL DEFAULT '',
    reasoning_mandatory boolean NOT NULL DEFAULT false,
    refreshed_at timestamptz NOT NULL,
    PRIMARY KEY (provider, base_url, model)
);
