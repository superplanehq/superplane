BEGIN;

CREATE TABLE mcp_api_tokens (
  id uuid NOT NULL DEFAULT uuid_generate_v4(),
  name text NOT NULL,
  user_id uuid NOT NULL,
  organization_id uuid NOT NULL,
  factory_id uuid NOT NULL,
  resource text NOT NULL,
  scopes jsonb NOT NULL DEFAULT '[]'::jsonb,
  token_hash text NOT NULL,
  created_at timestamp NOT NULL,
  last_used_at timestamp,

  PRIMARY KEY (id)
);

CREATE UNIQUE INDEX index_mcp_api_tokens_on_token_hash ON mcp_api_tokens (token_hash);

CREATE INDEX index_mcp_api_tokens_on_organization_id_and_factory_id ON mcp_api_tokens (organization_id, factory_id);

COMMIT;
