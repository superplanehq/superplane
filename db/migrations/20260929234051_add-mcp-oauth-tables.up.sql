BEGIN;

CREATE TABLE mcp_oauth_clients (
  id uuid NOT NULL DEFAULT uuid_generate_v4(),
  client_id text NOT NULL,
  client_name text NOT NULL DEFAULT '',
  redirect_uris jsonb NOT NULL DEFAULT '[]'::jsonb,
  created_at timestamp NOT NULL,

  PRIMARY KEY (id)
);

CREATE UNIQUE INDEX index_mcp_oauth_clients_on_client_id ON mcp_oauth_clients (client_id);

CREATE TABLE mcp_oauth_codes (
  id uuid NOT NULL DEFAULT uuid_generate_v4(),
  code_hash text NOT NULL,
  client_id text NOT NULL,
  redirect_uri text NOT NULL,
  resource text NOT NULL,
  code_challenge text NOT NULL,
  code_challenge_method text NOT NULL,
  user_id uuid NOT NULL,
  organization_id uuid NOT NULL,
  factory_id uuid NOT NULL,
  scopes jsonb NOT NULL DEFAULT '[]'::jsonb,
  expires_at timestamp NOT NULL,
  created_at timestamp NOT NULL,

  PRIMARY KEY (id)
);

CREATE UNIQUE INDEX index_mcp_oauth_codes_on_code_hash ON mcp_oauth_codes (code_hash);

CREATE TABLE mcp_oauth_refresh_tokens (
  id uuid NOT NULL DEFAULT uuid_generate_v4(),
  token_hash text NOT NULL,
  client_id text NOT NULL,
  user_id uuid NOT NULL,
  organization_id uuid NOT NULL,
  factory_id uuid NOT NULL,
  resource text NOT NULL,
  scopes jsonb NOT NULL DEFAULT '[]'::jsonb,
  expires_at timestamp NOT NULL,
  created_at timestamp NOT NULL,

  PRIMARY KEY (id)
);

CREATE UNIQUE INDEX index_mcp_oauth_refresh_tokens_on_token_hash ON mcp_oauth_refresh_tokens (token_hash);

COMMIT;
