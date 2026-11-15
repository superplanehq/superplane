CREATE TABLE organization_domains (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id     uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    domain              text NOT NULL,
    verification_token  text NOT NULL,
    verified_at         timestamptz,
    verification_method text,
    verified_by         uuid REFERENCES accounts(id),
    created_by          uuid REFERENCES accounts(id),
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT organization_domains_method_check
        CHECK (verification_method IS NULL OR verification_method IN ('dns', 'admin'))
);

CREATE UNIQUE INDEX organization_domains_org_domain
    ON organization_domains (organization_id, domain);

CREATE UNIQUE INDEX organization_domains_verified_domain
    ON organization_domains (domain) WHERE verified_at IS NOT NULL;

CREATE TABLE sso_connections (
    id                       uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id          uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    protocol                 text NOT NULL CHECK (protocol IN ('oidc')),
    preset                   text NOT NULL DEFAULT 'generic',
    display_name             text NOT NULL,
    status                   text NOT NULL DEFAULT 'draft'
                             CHECK (status IN ('draft', 'active', 'disabled')),
    enforcement              text NOT NULL DEFAULT 'optional'
                             CHECK (enforcement IN ('optional', 'required')),
    allow_owner_bypass       boolean NOT NULL DEFAULT true,
    jit_enabled              boolean NOT NULL DEFAULT true,
    default_role             text NOT NULL DEFAULT 'org_operator',
    max_session_age_seconds  integer NOT NULL DEFAULT 86400,
    idp_logout_enabled       boolean NOT NULL DEFAULT false,
    config                   jsonb NOT NULL DEFAULT '{}'::jsonb,
    client_secret            bytea,
    last_tested_at           timestamptz,
    last_test_result         jsonb,
    created_by               uuid REFERENCES accounts(id),
    created_at               timestamptz NOT NULL DEFAULT now(),
    updated_at               timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT sso_connections_preset_check
        CHECK (preset IN ('okta', 'entra', 'generic')),
    CONSTRAINT sso_connections_session_age_check
        CHECK (max_session_age_seconds BETWEEN 900 AND 604800)
);

CREATE UNIQUE INDEX sso_connections_one_per_org ON sso_connections (organization_id);

CREATE TABLE sso_identities (
    id                   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    connection_id        uuid NOT NULL REFERENCES sso_connections(id) ON DELETE CASCADE,
    organization_id      uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    account_id           uuid NOT NULL REFERENCES accounts(id),
    user_id              uuid REFERENCES users(id),
    subject              text,
    external_id          text,
    scim_user_name       text,
    email                text NOT NULL,
    name                 text,
    active               boolean NOT NULL DEFAULT true,
    provisioned_by       text NOT NULL CHECK (provisioned_by IN ('jit', 'scim')),
    idp_session_id       text,
    id_token             bytea,
    last_login_at        timestamptz,
    sessions_revoked_at  timestamptz,
    created_at           timestamptz NOT NULL DEFAULT now(),
    updated_at           timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX sso_identities_subject
    ON sso_identities (connection_id, subject) WHERE subject IS NOT NULL;

CREATE UNIQUE INDEX sso_identities_external_id
    ON sso_identities (connection_id, external_id) WHERE external_id IS NOT NULL;

CREATE UNIQUE INDEX sso_identities_account
    ON sso_identities (connection_id, account_id);

CREATE INDEX sso_identities_connection_email
    ON sso_identities (connection_id, lower(email));

CREATE INDEX sso_identities_connection_user_name
    ON sso_identities (connection_id, lower(scim_user_name));

CREATE INDEX sso_identities_idp_session
    ON sso_identities (connection_id, idp_session_id) WHERE idp_session_id IS NOT NULL;

CREATE TABLE scim_tokens (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    connection_id  uuid NOT NULL REFERENCES sso_connections(id) ON DELETE CASCADE,
    token_hash     text NOT NULL UNIQUE,
    description    text,
    created_by     uuid REFERENCES accounts(id),
    created_at     timestamptz NOT NULL DEFAULT now(),
    last_used_at   timestamptz,
    revoked_at     timestamptz
);

CREATE INDEX scim_tokens_connection_active
    ON scim_tokens (connection_id) WHERE revoked_at IS NULL;

ALTER TABLE users ADD COLUMN provisioned_by text
    CHECK (provisioned_by IN ('jit', 'scim'));
