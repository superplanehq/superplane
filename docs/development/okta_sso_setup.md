# Okta SSO & SCIM — developer setup (partial implementation)

This note matches the **first implementation slice** in the repo: **database**, **SCIM Users
API**, **managed accounts**, and **UI** hooks. **OIDC login** and **admin APIs** to configure
Okta from the app are **not** implemented yet; configure the org row manually (or via SQL) for
now.

See the product spec: [docs/prd/okta-sso-and-provisioning.md](../prd/okta-sso-and-provisioning.md).

## 1. Migrate databases

```bash
make db.migrate DB_NAME=superplane_dev
make db.migrate DB_NAME=superplane_test
```

## 2. Enable SCIM for an organization

1. Generate a long random bearer secret (e.g. 32+ bytes), **UTF-8 string**.
2. Compute **SHA-256** hex of that string (same as API tokens: `crypto.HashToken` in Go).
3. Insert or update **`organization_okta_idp`**:

- **`organization_id`**: target org UUID.
- **`issuer_base_url`**: Okta org URL, e.g. `https://dev-12345.okta.com` (no trailing path).
- **`oauth_client_id`**: placeholder allowed until OIDC is wired (`''` not allowed — use a
  dummy value if needed).
- **`oauth_client_secret_ciphertext`**: nullable until OIDC is wired.
- **`scim_bearer_token_hash`**: hex from step 2.
- **`scim_enabled`**: `true`.
- **`oidc_enabled`**: `false` until OIDC ships.

**Okta SCIM base URL** (this codebase):

```text
https://<your-superplane-host>/api/v1/scim/<organization_uuid>/v2/
```

Examples:

- `GET .../v2/ServiceProviderConfig`
- `POST .../v2/Users` with `Authorization: Bearer <raw secret from step 1>`

## 3. Managed accounts (SCIM)

Users created via SCIM get **`accounts.managed_account = true`**. They **cannot** create new
organizations (API **403** and UI hides the flow).

## 4. Next steps (not done yet)

- Per-org **OIDC** authorize/callback using **`coreos/go-oidc`** and **`x/oauth2`**.
- **gRPC / REST** to upsert **`organization_okta_idp`** (secrets encrypted with **`pkg/crypto`**).
- Sync **`organizations.allowed_providers`** with **`okta`** when OIDC is enabled.
