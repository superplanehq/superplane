# Okta SSO & SCIM — developer setup

Covers **SCIM**, **admin IdP settings**, **OIDC browser login** (authorization code + ID token
verification), and **managed accounts**. See
[docs/prd/okta-sso-and-provisioning.md](../prd/okta-sso-and-provisioning.md) for product
context.

## 1. Migrate databases

```bash
make db.migrate DB_NAME=superplane_dev
make db.migrate DB_NAME=superplane_test
```

## 2. Admin API — `organization_okta_idp`

Authenticated org admin with **org read/update**:

- `GET /api/v1/organizations/{id}/okta-idp` — current settings (no secrets).
- `PATCH /api/v1/organizations/{id}/okta-idp` — issuer (**https** URL), OAuth client id/secret
  (encrypted), `oidc_enabled`, `scim_enabled`. Turning **`oidc_enabled` on** requires a configured
  client secret and appends **`okta`** to **`organizations.allowed_providers`** if missing.
- `POST /api/v1/organizations/{id}/okta-idp/rotate-scim-token` — returns the **plaintext** SCIM
  bearer token **once**; the hash is stored automatically.

## 3. SCIM

**Base URL:**

```text
https://<your-superplane-host>/api/v1/scim/<organization_uuid>/v2/
```

Enable **`scim_enabled`** via the PATCH above after rotating a SCIM token.

## 4. OIDC browser login

**Okta application settings**

- **Sign-in redirect URI**:
  `https://<your-superplane-host>/auth/okta/<organization_uuid>/callback`
- **Sign-out** (optional): same host as your app.
- Scopes used: `openid`, `email`, `profile`.

**Start URL** (send users here after they pick their org, or deep-link):

```text
https://<your-superplane-host>/auth/okta/<organization_uuid>?redirect=%2F
```

`redirect` must be a same-origin path (e.g. `%2F` or `%2Fsettings`).

**Flow**

1. Server checks org **`allowed_providers`** includes **`okta`**, **`oidc_enabled`**, and a stored
   client secret.
2. User is redirected to Okta’s authorize endpoint (from OIDC discovery on **`issuer_base_url`**).
3. **`state`** is a short-lived signed JWT (CSRF + org id + redirect).
4. Callback exchanges the code, verifies the **ID token** only (`go-oidc`), then:
   - Resolves an **active** org **`users`** row by normalized **email** from the ID token.
   - Requires a row in **`organization_scim_user_mappings`** for that user (no JIT from OIDC
     alone).
   - Upserts **`account_providers`** for **`okta`** (stable **`provider_id`** from issuer + `sub`).
5. Sets the normal **`account_token`** session cookie and redirects.

## 5. Managed accounts (SCIM)

Users created via SCIM get **`accounts.managed_account = true`**. They **cannot** create new
organizations (API **403** and UI hides the flow).

## 6. Follow-ups

- Optional **nonce** / **PKCE** hardening for confidential clients.
- **IdP-initiated** login (deferred in PRD).
