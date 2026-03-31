ALTER TABLE organization_okta_idp
    ADD COLUMN issuer_base_url                TEXT NOT NULL DEFAULT '',
    ADD COLUMN oauth_client_id                TEXT NOT NULL DEFAULT '',
    ADD COLUMN oauth_client_secret_ciphertext BYTEA,
    ADD COLUMN oidc_enabled                   BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE organization_okta_idp
    DROP COLUMN saml_idp_sso_url,
    DROP COLUMN saml_idp_issuer,
    DROP COLUMN saml_idp_certificate_pem,
    DROP COLUMN saml_enabled;
