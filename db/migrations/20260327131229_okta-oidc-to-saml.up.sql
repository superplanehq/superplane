ALTER TABLE organization_okta_idp
    ADD COLUMN saml_idp_sso_url         TEXT NOT NULL DEFAULT '',
    ADD COLUMN saml_idp_issuer           TEXT NOT NULL DEFAULT '',
    ADD COLUMN saml_idp_certificate_pem  TEXT NOT NULL DEFAULT '',
    ADD COLUMN saml_enabled              BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE organization_okta_idp
    DROP COLUMN issuer_base_url,
    DROP COLUMN oauth_client_id,
    DROP COLUMN oauth_client_secret_ciphertext,
    DROP COLUMN oidc_enabled;
