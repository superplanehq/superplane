CREATE TABLE integration_model_catalogs (
    integration_id uuid PRIMARY KEY REFERENCES app_installations(id) ON DELETE CASCADE,
    models jsonb NOT NULL DEFAULT '[]'::jsonb,
    observed_at timestamp with time zone NOT NULL
);

-- Organization selections were also used as a cache. They are not provider
-- catalogs and must not become workspace restrictions or availability data.
DROP TABLE organization_byok_model_allowlists;
DROP TABLE organization_hosted_model_allowlists;
