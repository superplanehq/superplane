## Okta integration (per organization)

Superplane supports configuring Okta SSO and SCIM **per organization**, similar to Semaphore.

This section documents only the **settings API** you now have; the SAML and SCIM endpoints build on top of this and can be wired from Okta as needed.

### HTTP API endpoints

All endpoints are authenticated with the normal Superplane session cookie and require the caller to be **Org Owner or Org Admin** in the target organization.

- `GET /organizations/{orgId}/okta`
  - Returns the current Okta configuration for the organization.
  - Response:
    ```json
    {
      "saml_issuer": "https://example.okta.com/app/abc123/sso/saml",
      "saml_certificate": "-----BEGIN CERTIFICATE-----...",
      "enforce_sso": false,
      "has_scim_token": true
    }
    ```

- `PUT /organizations/{orgId}/okta`
  - Creates or updates the Okta configuration.
  - Request body:
    ```json
    {
      "saml_issuer": "https://example.okta.com/app/abc123/sso/saml",
      "saml_certificate": "-----BEGIN CERTIFICATE-----...",
      "enforce_sso": true
    }
    ```
  - Does **not** return the SCIM token or its hash.

- `POST /organizations/{orgId}/okta/scim-token`
  - Rotates the SCIM token for the organization.
  - Requires that Okta settings already exist for the org.
  - Response (token is returned **once**):
    ```json
    {
      "token": "base64-url-random-token"
    }
    ```
  - The hash of this token is stored in the database; the clear text value is not stored server-side.

### SAML ACS endpoint

For each organization, the SAML Assertion Consumer Service (ACS) endpoint is:

- `POST {BASE_URL}/orgs/{orgId}/okta/auth`

Where:

- `{BASE_URL}` is the public base URL of your Superplane deployment.
- `{orgId}` is the UUID of the organization in Superplane.

This endpoint:

- Validates the SAML response using:
  - Issuer from `saml_issuer`.
  - Certificate from `saml_certificate`.
- Extracts the `email` attribute from the assertion.
- Finds the matching Account and User in the target organization.
- Issues a standard `account_token` session cookie and redirects to `/{orgId}`.

### Notes

- Okta configuration is **per organization**; there is no global Okta provider.
- The SCIM token is only exposed once at creation/rotation time; clients must store it securely.
- The SAML implementation is based on `gosaml2` and expects a standard Okta SAML app with an `email` attribute.

