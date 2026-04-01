BEGIN;

CREATE TABLE organization_okta_idp (
    id                              uuid NOT NULL DEFAULT uuid_generate_v4(),
    organization_id                 uuid NOT NULL,
    issuer_base_url                 character varying(512) NOT NULL,
    oauth_client_id                 character varying(512) NOT NULL,
    oauth_client_secret_ciphertext  bytea,
    oidc_enabled                    boolean NOT NULL DEFAULT false,
    scim_bearer_token_hash          character varying(64),
    scim_enabled                    boolean NOT NULL DEFAULT false,
    created_at                      timestamp without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at                      timestamp without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT organization_okta_idp_pkey PRIMARY KEY (id),
    CONSTRAINT organization_okta_idp_organization_id_key UNIQUE (organization_id),
    CONSTRAINT organization_okta_idp_organization_id_fkey
        FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE
);

CREATE INDEX idx_organization_okta_idp_organization_id ON organization_okta_idp(organization_id);

CREATE TABLE organization_scim_user_mappings (
    id               uuid NOT NULL DEFAULT uuid_generate_v4(),
    organization_id  uuid NOT NULL,
    user_id          uuid NOT NULL,
    external_id      text,
    created_at       timestamp without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       timestamp without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT organization_scim_user_mappings_pkey PRIMARY KEY (id),
    CONSTRAINT organization_scim_user_mappings_organization_id_user_id_key
        UNIQUE (organization_id, user_id),
    CONSTRAINT organization_scim_user_mappings_organization_id_fkey
        FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    CONSTRAINT organization_scim_user_mappings_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX organization_scim_user_mappings_org_external_id_uidx
    ON organization_scim_user_mappings (organization_id, external_id)
    WHERE external_id IS NOT NULL;

CREATE INDEX idx_organization_scim_user_mappings_organization_id
    ON organization_scim_user_mappings(organization_id);
CREATE INDEX idx_organization_scim_user_mappings_user_id
    ON organization_scim_user_mappings(user_id);

ALTER TABLE accounts
    ADD COLUMN managed_account boolean NOT NULL DEFAULT false;

COMMIT;
