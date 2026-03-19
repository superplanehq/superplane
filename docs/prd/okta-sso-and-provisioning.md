# Okta SSO (OIDC) and User Provisioning

## Overview

This PRD defines support for **Okta** as an identity source for SuperPlane: browser-based
sign-in via **OpenID Connect (OIDC)** and **directory-driven user lifecycle** via **SCIM 2.0**
(create, update, deactivate users). The technical baseline follows Okta’s guidance for building
an SSO integration, including use of the **org authorization server**, **Authorization Code flow
with a client secret** (web application pattern), and **token handling** appropriate for that
server type. See Okta’s [Build a Single Sign-On (SSO) integration (OpenID
Connect)](https://developer.okta.com/docs/guides/build-sso-integration/openidconnect/main/).

Today, SuperPlane authenticates humans through OAuth providers wired in `pkg/authentication`
(Goth + GitHub/Google), optional password auth, and links identities to `accounts` /
`account_providers`. Organizations already constrain allowed IdPs via
`organizations.allowed_providers`. Okta support extends that model so **each customer
organization** can connect **their** Okta tenant (issuer, client credentials, redirect URIs)
rather than relying on a single global Okta app for all tenants.

## Problem Statement

Enterprise customers standardize on Okta for workforce identity. Without Okta:

- Users cannot sign in with corporate policies (MFA, session length, conditional access)
  enforced by Okta.
- IT cannot provision and deprovision SuperPlane access from Okta’s lifecycle and group rules.
- Admins fall back to manual invitations and provider sprawl (personal GitHub/Google), which
  weakens governance.

## Goals

1. **OIDC login**: End users can sign in to SuperPlane through Okta using OIDC (Authorization
   Code flow, confidential web client), consistent with Okta’s recommended web-app pattern.
2. **Per-organization configuration**: Each SuperPlane organization configures **at most one**
   Okta integration (issuer / Okta domain, client ID, client secret, allowed redirect URIs).
3. **Correct token practices**: Validate **ID tokens** with Okta’s JWKS and respect **key
   rotation**. Do **not** rely on access tokens for SuperPlane session creation (we do not call
   Okta resource APIs); keep processing minimal (see Decisions).
4. **User provisioning (SCIM 2.0)**: Okta (or another SCIM client) can provision users into the
   correct SuperPlane organization, update profile attributes, and deactivate users when
   assignments end.
5. **Admin UX**: Clear setup documentation and in-app (or admin-console) steps: redirect URIs,
   scopes, SCIM endpoint URL, and SCIM bearer token rotation.
6. **Managed workforce users**: Identities introduced through **SCIM** (and thus Okta) are
   **marked as managed accounts** and stay inside their employer’s SuperPlane organization — they
   **must not** create new organizations or use self-serve “start your own org” flows (see
   **Decisions**).

## Non-Goals (Initial Release)

- **Okta Integration Network (OIN) submission** as a blocking deliverable — implementation should
  follow OIN-oriented OIDC constraints so a future OIN submission is feasible, but packaging,
  wizard artifacts, and Okta verification are out of scope unless explicitly scheduled.
- **SPAs talking to Okta directly** for token exchange — browser clients continue to use the
  server-side authorization code flow; tokens are not managed in frontend storage beyond
  existing session patterns.
- **Custom authorization servers** (`/oauth2/{authorizationServerId}/...`) as the default path
  for customer deployments covered by this PRD — default documentation and validators target the
  **org authorization server** (`/oauth2/v1/...`, issuer `https://{oktaDomain}`) per Okta’s
  OIN-oriented guidance. (Some customers may use custom servers; supporting that explicitly can be
  a follow-up if product requirements demand it.)
- **Refresh tokens / `offline_access`** where product policy or OIN constraints disallow them —
  session length and re-auth behavior should be documented explicitly.
- **Full group sync into SuperPlane RBAC groups** — optional later phase unless we define a
  minimal mapping (e.g., SCIM groups → SuperPlane groups) in a separate iteration.
- **SAML** — this initiative is **OIDC only**; SAML is out of scope unless raised as a separate
  project.
- **IdP-initiated login** (initiate login URI / dashboard-launched flows) — **deferred** past
  v1.

## Primary Users

- **Org admins / IT**: Configure Okta OIDC app, SCIM provisioning, and troubleshoot login.
- **End users**: Sign in with Okta through the normal SuperPlane login experience.
- **Security / compliance**: Rely on Okta for MFA, audit, and offboarding via SCIM deactivate.

## Okta Alignment (Authoritative Constraints)

The following constraints from Okta’s OIDC SSO guide should be treated as **design requirements**
unless we explicitly document a deviation:

- **Flow**: **Authorization Code** with **client secret** (web application).
- **Authorization server**: Prefer **org authorization server** endpoints (not custom server
  paths) for customer setups aligned with OIN guidance.
- **Scopes**: Request **`openid`** plus standard OIDC scopes needed for profile/email (e.g.
  `profile`, `email`) as allowed by the customer’s Okta policies.
- **Key rotation**: Periodically refresh JWKS from Okta’s **`/keys`** (or discovery document);
  tolerate rotation failures by refreshing keys.
- **Access token**: Okta still returns an access token with the code exchange; SuperPlane does
  not need it for identity or for calling Okta APIs in v1 — **no introspection or JWT validation
  of the access token** unless a later feature requires it (see Decisions).
- **Multi-tenancy**: **One Okta app (issuer + client credentials) per SuperPlane organization** —
  no shared global Okta client for all enterprises.
- **Rate limits**: Outbound calls to Okta (authorize redirect is user-driven; token exchange,
  JWKS, SCIM) should respect [Okta rate
  limits](https://developer.okta.com/docs/reference/rate-limits/); use caching where safe (e.g.
  JWKS).

## User Stories

### Authentication

1. As an org admin, I can enable Okta SSO for my SuperPlane organization and provide Okta’s
   issuer URL, client ID, and client secret.
2. As an end user, I can choose “Sign in with Okta” and complete login through my company’s
   Okta, then land in the correct SuperPlane org.
3. As an org admin, I can disable Okta SSO or rotate credentials without breaking other orgs.

### Provisioning

4. As an org admin, I can enable SCIM provisioning and copy a **base URL** and **SCIM bearer
   token** into Okta’s provisioning settings.
5. As an IT admin, when I assign the SuperPlane app in Okta, SCIM **creates** the user in the
   correct SuperPlane organization before they can sign in with Okta.
6. As an IT admin, when I unassign or deactivate a user in Okta, SuperPlane reflects deactivation
   (cannot sign in; existing sessions invalidated or expired per policy).

### Governance

7. As a security stakeholder, **managed** (SCIM-provisioned) accounts **cannot** create
   additional SuperPlane organizations; only **unmanaged** self-serve accounts retain that
   ability (until policy changes).

## Architecture (High Level)

### OIDC (Login)

1. **Configuration**: **one row per org** in **`organization_okta_idp`** (see **Database**). Holds
   issuer, OAuth client fields, SCIM bearer hash, flags. IdP-initiated / initiate-login URI:
   **deferred** (see Decisions).
2. **Login start**: User selects Okta; backend builds authorize URL against
   `https://{oktaDomain}/oauth2/v1/authorize` with `response_type=code`,
   `scope=openid email profile` (exact set TBD), `state`, and PKCE if we adopt it (recommended
   for public clients; confidential web clients often omit PKCE but may still use it for
   hardening).
3. **Callback**: Exchange code at `https://{oktaDomain}/oauth2/v1/token` with client secret;
   validate **ID token** only (iss, aud, exp, nonce if used). **Ignore** the access token for v1
   identity/session purposes (do not introspect or verify it unless a future feature needs it).
4. **Account linking**: Map Okta `sub` (and issuer) to `account_providers` with a stable provider
   key (e.g. `okta` + issuer fingerprint, or a single `okta` provider with composite uniqueness on
   `(issuer, sub)` — detailed schema TBD). Reuse `findOrCreateAccountForProvider` patterns from
   `pkg/authentication/authentication.go` where possible.
5. **Organization resolution**: **SCIM before first login.** A user must already exist in the
   target SuperPlane org via SCIM; OIDC login **links** `sub` to that user (e.g. match
   `userName` / email to the SCIM-provisioned record). No JIT org membership from OIDC alone.

### SCIM 2.0 (Provisioning)

1. **Per-org SCIM endpoint** (example shape): `POST/PATCH/DELETE` under a path scoped by org
   identifier and authenticated by a **hashed bearer token** stored encrypted (similar patterns
   to integration secrets).
2. **Supported resources** (minimum): **Users** — `POST` create, `GET` retrieve/list (if required
   by Okta tester), `PATCH` update, `PUT` optional, `DELETE` or `PATCH` active=false for
   deactivate.
3. **Id mapping**: SCIM **`id`** returned to Okta is **`users.id`**. Optional IdP **`externalId`**
   lives in **`organization_scim_user_mappings`** (not on `users`). OIDC first login attaches
   `sub` via `account_providers` when `userName` / email matches the SCIM-created user.

**Note:** SCIM is not part of the linked OIDC guide but is the standard way Okta provisions
users into SaaS apps and is in scope for this PRD.

### Identity: do workforce users need `accounts`?

**For v1, yes.** SuperPlane’s browser session is **account-centric** today: the **`account_token`**
cookie carries a JWT whose **`sub` is `accounts.id`**, and `authenticateUserByCookie` loads the
**`accounts`** row, then resolves the active **`users`** row for the current org using **account
email** plus **`x-organization-id`** (see `pkg/public/middleware/auth.go`). **`account_providers`**
(including Okta) also hangs off **`accounts`**. **`managed_account`** (see **Database**) marks
directory-provisioned identities on **`accounts`**.

Conceptually a directory-only person might be “just an org member,” but **dropping `accounts`**
would mean redesigning session issuance, middleware resolution, provider linkage, and several HTTP
entry points — a **large auth project**, not an Okta add-on. **SCIM therefore keeps creating
`accounts` + `users`** (thin global identity + org membership), same as the invitation path.

**Future (non-commitment):** a **`(organization_id, user_id)`**-scoped session and provider rows
keyed off **`users`** could reduce global state; track as follow-up if product wants true
org-local identities without a shared account.

## Database schema

Use `make db.migration.create NAME=<name>` for all DDL (see `AGENTS.md`); this section is the
intended shape only.

### New table: `organization_okta_idp`

One row per SuperPlane organization (matches **one Okta tenant per org**). Suggested columns:

- **`id`**: primary key UUID.
- **`organization_id`**: FK → `organizations.id`, **UNIQUE** (one Okta connection per org).
- **`issuer_base_url`**: normalized Okta issuer base (e.g. `https://dev-xxx.okta.com`) for
  discovery and ID token `iss` checks.
- **`oauth_client_id`**: OIDC client identifier from the customer’s Okta app.
- **`oauth_client_secret_encrypted`**: ciphertext from `pkg/crypto` `Encryptor`; never store
  plaintext.
- **`oidc_enabled`**: whether org admins turned on Okta sign-in.
- **`scim_bearer_token_hash`**: hash of the SCIM bearer secret (same idea as `users.token_hash`,
  e.g. SHA-256 of raw token). Nullable until SCIM is configured.
- **`scim_enabled`**: whether SCIM is accepted for this org.
- **`created_at`, `updated_at`**: audit timestamps.

Optional later: `last_scim_request_at`, rotation metadata for secrets, encrypted backup of SCIM
token if you ever need “show once” UX (prefer hash-only if rotation is easy).

### `users`

- **No schema changes** on **`users`** for Okta or SCIM.

SCIM **User create** still follows the same pattern as an accepted invitation: create
**`accounts`** + **`users`** with a real **`account_id`**, then first Okta login adds
**`account_providers`** (see below).

### New table: `organization_scim_user_mappings`

Auxiliary rows for **SCIM-provisioned** humans (one per `(organization_id, user_id)`):

- **`organization_id`**, **`user_id`** — FKs; **UNIQUE** together so each org member has at most
  one SCIM mapping row.
- **`external_id`** (`text`, nullable) — SCIM **`externalId`** from the IdP when Okta sends it.
  **Partial UNIQUE** on `(organization_id, external_id)` **WHERE** `external_id IS NOT NULL`.
- **`created_at`, `updated_at`** — audit.

This is **not** a second user store: it only holds IdP-specific identifiers. The SCIM **`id`**
in API responses remains **`users.id`**. If product later decides `externalId` is unnecessary,
this table can stay minimal or merge into another artifact — but **do not** add columns to
**`users`** for that purpose.

### `accounts` (extend)

- **`managed_account`** (`boolean`, **NOT NULL**, default **`false`**) — set **`true`** when the
  **account row is created by SCIM** (workforce / directory-managed identity). These accounts
  **cannot** create new SuperPlane organizations: enforce in **`createOrganization`**
  (`pkg/public/server.go` today: any logged-in account can create an org) and in the **web UI**
  (hide or disable “Create organization”). **Break-glass** (support clearing the flag) is
  optional unless ops asks for it.

Self-serve accounts (GitHub/Google signup, password signup where allowed) stay **unmanaged**
  (**`managed_account = false`**).

**Today (without this work)**: any authenticated account can create a new organization; directory
users would inherit that unless we add this guard.

### `account_providers` (behavior + constants)

- Add provider constant **`okta`** (alongside `github`, `google` in `pkg/models`).
- **`provider_id`** is `varchar(255)` with a **global** unique `(provider, provider_id)` today.
  **Encode issuer + Okta `sub`** as a short deterministic string (e.g. **64-char hex SHA-256**
  of canonical issuer, a delimiter, and `sub`) so tenants cannot collide on the same `sub`.
- **`UNIQUE (account_id, provider)`** still allows one Okta link per account; that matches
“single Okta IdP per SuperPlane org” for a given person.

### `organizations`

- No new columns required if **`allowed_providers`** (`JSON` slice) already lists permitted IdPs:
  **add `okta`** when Okta OIDC is enabled (and remove when disabled), same pattern as GitHub /
  Google.
- Alternatively, derive “Okta allowed” from **`organization_okta_idp.oidc_enabled`** in code;
  still keep `allowed_providers` consistent so existing checks like `IsProviderAllowed` stay
  truthful.

### OIDC `state` / CSRF (not a durable table requirement)

Short-lived **server-side session or cache** entry (Redis, encrypted cookie payload, or DB table
with TTL) mapping `state` → `organization_id` + nonce for the authorize redirect. If you use a
table, treat it as ephemeral with periodic cleanup — document in detailed design.

### What we are not adding (v1)

- A **`scim_users`** shadow table that duplicates name/email/profile — **`users`** remains the
  system of record; **`organization_scim_user_mappings`** is only IdP linkage metadata.
- **SAML** metadata tables.
- **Group** resource persistence for SCIM (non-goal until group sync is in scope).

## SuperPlane Codebase Touchpoints (Expected)

- **`pkg/authentication`**: Okta-specific OIDC flow alongside existing Goth providers (see
  **Technologies** — prefer `go-oidc` + `x/oauth2`, not Goth, for per-org issuers).
- **`pkg/public/server.go`**: Wire org-based Okta config (one per org); callback routes may need
  org id in path or state. **Reject organization creation** when **`accounts.managed_account`** is
  **true** (repeat on any future org-creation API).
- **`pkg/models`**: `organization_okta_idp`; `organization_scim_user_mappings`; **`accounts`**
  `managed_account`; provider constants.
- **`organizations.allowed_providers`**: Include an `okta` (or OIDC-generic) entry when enabled.
- **`web_src`**: Okta login entry; SSO/SCIM settings; org-switcher / create-org UX respects
  **`managed_account`** (no self-serve new org when **true**).
- **Authorization / security docs**: Document token validation, secret encryption, and SCIM auth.

## Technologies

Selections below favor **what the repo already uses** and add **small, well-maintained** libraries
where OIDC requires JWKS and discovery. Line items marked **(new)** are not in `go.mod` today and
require a deliberate dependency add.

### Runtime, routing, and data

- **Go** (module at `go 1.25`) — same server binary as the rest of SuperPlane.
- **`github.com/gorilla/mux`** — register Okta authorize/callback routes and SCIM routes next to
  existing `pkg/authentication` and `pkg/public` wiring.
- **PostgreSQL + GORM** (`gorm.io/gorm`, `gorm.io/driver/postgres`) — persist per-org Okta issuer,
  client id, encrypted client secret, SCIM token material, and SCIM↔user id mapping.
- **gRPC + gRPC-Gateway** — org-admin APIs for enabling SSO, rotating secrets, and surfacing
  read-only setup metadata should follow existing API patterns (proto → generated REST).

### OIDC: discovery, token exchange, and ID token verification

- **`golang.org/x/oauth2`** (already required) — Authorization Code exchange at Okta’s **token**
  endpoint (`client_secret` post), scoped config per org (`ClientID`, `ClientSecret`,
  `RedirectURL`, `Endpoint` derived from issuer).
- **`github.com/coreos/go-oidc/v3/oidc` (new)** — load issuer metadata via
  `/.well-known/openid-configuration`, construct an **ID token verifier** (audience, expiry,
  signature against Okta JWKS). This covers **JWKS fetch + rotation** without custom key-cache code
  beyond what the library provides.
- **Do not use `markbates/goth` for Okta v1** — Goth registers providers at process start with
  static endpoints; SuperPlane needs **one issuer + client pair per org**. Keep Goth for
  GitHub/Google unchanged.

### SuperPlane session vs. Okta tokens

- **`github.com/golang-jwt/jwt/v4`** (`pkg/jwt`) — continues to mint **SuperPlane session**
  cookies (HMAC, internal `sub`). This is unrelated to verifying Okta’s **RS256 ID tokens**; do
  not reuse the session `Signer` for Okta JWTs.

### Secret storage

- **`pkg/crypto` `Encryptor`** (AES-GCM in production) — encrypt Okta **client secret** and SCIM
  **bearer token** (or a hash of the token for comparison plus encrypted backup if needed) using
  the same patterns as other sensitive integration material.

### Outbound HTTP to Okta

- **`net/http`** with standard timeouts — sufficient for token exchange and discovery from the
  auth handler. Component/trigger code uses `core.HTTPContext`; auth can use a small dedicated
  `http.Client` with explicit `Transport` timeouts, or a shared internal helper if one exists for
  server-side outbound calls.

### SCIM 2.0 (server)

- **`github.com/elimity-com/scim` (new)** — yes, **use this** for the SCIM MVP. It implements SCIM
  v2 with **CRUD + PATCH**, **schema validation** before callbacks, built-in **`/Schemas`**,
  **`/ServiceProviderConfig`**, and **`/ResourceTypes`** (all commonly probed by IdPs), plus a
  **`filter`** package for list queries. We implement **`scim.ResourceHandler`** for **Users**
  only (create/get/replace/delete/patch) and wire org context + bearer auth **outside** the
  library (e.g. mux subrouter per org path prefix, then `StripPrefix` into the SCIM `http.Handler`
  from `scim.NewServer`). See [elimity-com/scim](https://github.com/elimity-com/scim).
- **Caveats** (from upstream README): project describes itself as **early stage**; **minor
  version bumps may change APIs** — pin versions and read release notes. **Bulk, sorting**, and
  other optional RFC features are **not** supported; confirm Okta’s provisioning profile does
  not require them for v1. Immutable / writeOnly attribute rules need **explicit checks** in our
  handlers where the library cannot enforce them alone.
- **Fallback**: if adoption blocks on routing or Okta quirks, narrow hand-rolled endpoints remain
  possible for a subset of operations; prefer staying on elimity-com/scim while we extend tests.

### Web UI

- **Vite + React** (`web_src/`) — Okta sign-in when the org allows it; SSO/SCIM admin settings;
  **hide or disable “Create organization”** for **managed** accounts. Regenerate OpenAPI/TS
  client after proto changes (`make` targets in `AGENTS.md`).

### Observability and quality

- **OpenTelemetry** (`otelmux` already wraps the router) — extend spans or attributes on new auth
  and SCIM routes as needed; avoid logging secrets or raw tokens.
- **`github.com/stretchr/testify`** — unit tests for handlers, token verification errors, and SCIM
  edge cases; **`net/http/httptest`** for handler integration tests.

## Security Considerations

- **Secrets**: Client secret and SCIM bearer token only stored encrypted at rest; never returned
  in APIs after write.
- **CSRF / state**: `state` parameter must bind to session and org intent; prevent login CSRF and
  cross-org confusion.
- **SCIM**: Constant-time comparison of bearer tokens; rate limit SCIM routes; strong unguessable
  tokens; rotation workflow.
- **Deprovisioning**: Define whether `DELETE` removes the user row or soft-deactivates; impact
  on audit logs and ownership transfer.
- **Email trust**: Do not treat `email_verified` as sole proof of ownership where policy
  requires; Okta documents limitations for some claims in the same OIDC SSO guide linked in
  Overview.
- **Managed accounts**: **`managed_account = true`** blocks self-serve org creation; enforce
  **server-side** so UI-only hiding is not the sole control.

## Implementation Phases (Suggested)

1. **Design lock**: Data model for per-org OIDC + SCIM; account linking rules; session/org
   binding.
2. **SCIM MVP** (before OIDC): Users resource + bearer auth + Okta provisioning tester
   compatibility — required so users exist before first Okta login.
3. **OIDC MVP**: Read-only admin config, authorize + callback + token exchange, ID token
   validation, login path that **links** to SCIM-provisioned users, `allowed_providers`
   integration.
4. **Hardening**: JWKS caching with rotation, metrics and structured errors for IT admins.
5. **UX & docs**: Admin setup guide mirroring Okta’s admin console fields (redirect URIs, assign
   users, SCIM credentials).

## Decisions

1. **Org binding / first login**: Require **prior SCIM provisioning** — users must exist in the
   SuperPlane org via SCIM before Okta OIDC login succeeds (link `sub` to that user; no JIT from
   OIDC alone).
2. **Provider granularity**: **Exactly one Okta (OIDC) configuration per SuperPlane
   organization.** Other IdPs (e.g. Google) remain governed by existing `allowed_providers`
   rules separately; this PRD does not add multi-Okta or merged IdP priority within one org.
3. **Initiate login URI / IdP-initiated flows**: **Deferred** past v1.
4. **SAML**: **OIDC only** for this initiative; SAML is a non-goal (see Non-Goals).
5. **Access token**: **Simplify** — SuperPlane does not call Okta APIs with the user’s access
   token in v1; **validate the ID token only** and do not introspect or verify the access token
   unless a later requirement appears.
6. **Managed workforce accounts**: Accounts **created via SCIM** are marked **`managed_account`**
   (**`true`**). They cannot create new organizations; IT keeps them inside the provisioned org.
   Self-serve sign-ups remain **unmanaged** (**`managed_account = false`** default).
7. **Keep `accounts` for SCIM users (v1)**: Required to fit existing cookie auth and
   **`account_providers`**; org-only identity without accounts is **out of scope** (see
   **Architecture → Identity**).

## Success Metrics

- Successful end-to-end login via Okta in a test Okta org (Authorization Code, org auth server).
- Okta **Run Profile** / provisioning tests pass for create, update, deactivate (or equivalent
  manual QA checklist).
- No cross-tenant credential leakage in multi-org staging tests.
- **Managed** (SCIM-created) accounts receive **403** (or equivalent) from org-creation APIs and
  never see a working “new org” path in the UI.

## References

- [Okta: Build a Single Sign-On (SSO) integration — OpenID
  Connect](https://developer.okta.com/docs/guides/build-sso-integration/openidconnect/main/)
- [Okta: OAuth 2.0 and OpenID Connect overview][okta-oauth-overview] (supporting context)
- SCIM 2.0 ([RFC 7644](https://datatracker.ietf.org/doc/html/rfc7644)) for provisioning protocol
  semantics

[okta-oauth-overview]: https://developer.okta.com/docs/concepts/oauth-openid/
