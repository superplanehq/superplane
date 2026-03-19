# Okta SSO & SCIM — developer setup (partial implementation)

This note matches the current implementation slice: **database**, **SCIM Users API**, **managed
accounts**, **UI** hooks, and **REST/gRPC admin APIs** for **`organization_okta_idp`** (issuer,
OAuth client id/secret, OIDC/SCIM toggles, SCIM bearer rotation). **OIDC login** is not wired
yet; you can still enable **`oidc_enabled`** in settings for future use.

See the product spec: [docs/prd/okta-sso-and-provisioning.md](../prd/okta-sso-and-provisioning.md).

## 1. Migrate databases

```bash
make db.migrate DB_NAME=superplane_dev
make db.migrate DB_NAME=superplane_test
```

## 2. Enable SCIM for an organization

### Option A — API (recommended)

Authenticated org admin with **org read/update**:

- `GET /api/v1/organizations/{id}/okta-idp` — current settings (no secrets).
- `PATCH /api/v1/organizations/{id}/okta-idp` — create/update issuer (**https** URL), OAuth
  client id, optional client secret (encrypted), `oidc_enabled`, `scim_enabled`.
- `POST /api/v1/organizations/{id}/okta-idp/rotate-scim-token` — returns the **plaintext** SCIM
  bearer token **once**; store the hash server-side automatically.

Then set **`scim_enabled`** via the same PATCH when ready.

### Option B — SQL (manual)

1. Generate a long random bearer secret (e.g. 32+ bytes), **UTF-8 string**.
2. Compute **SHA-256** hex of that string (same as API tokens: `crypto.HashToken` in Go).
3. Insert or update **`organization_okta_idp`**:

- **`organization_id`**: target org UUID.
- **`issuer_base_url`**: Okta issuer URL, e.g. `https://dev-12345.okta.com/oauth2/default`.
- **`oauth_client_id`**: OAuth app client id (non-empty).
- **`oauth_client_secret_ciphertext`**: app-encrypted secret (or null until OIDC is wired).
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
- Sync **`organizations.allowed_providers`** with **`okta`** when OIDC is enabled.
