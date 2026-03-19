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

## Architecture (High Level)

### OIDC (Login)

1. **Configuration** (per SuperPlane `organization` or dedicated `organization_sso_connections`
   table — TBD in detailed design): **one row per org**, Okta domain/issuer, client ID, encrypted
   client secret, registered redirect URI(s). IdP-initiated / initiate-login URI: **deferred**
   (see Decisions).
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
3. **Id mapping**: Store Okta `externalId` ↔ SuperPlane user id. OIDC first login attaches
   `sub` to the existing SCIM-created user when identifiers align (e.g. `userName` / email).

**Note:** SCIM is not part of the linked OIDC guide but is the standard way Okta provisions
users into SaaS apps and is in scope for this PRD.

## SuperPlane Codebase Touchpoints (Expected)

- **`pkg/authentication`**: New Okta/OIDC provider or parallel implementation (Goth may not expose
  all knobs for dynamic per-org endpoints — evaluate Goth vs. `core.HTTPContext` + manual OIDC).
- **`pkg/public/server.go`**: Wire org-based Okta config (one per org); callback routes may need
  org id in path or state.
- **`pkg/models`**: SSO connection model; SCIM token storage; provider constants.
- **`organizations.allowed_providers`**: Include an `okta` (or OIDC-generic) entry when enabled.
- **`web_src`**: Login UI: Okta button when org or instance supports it; admin settings for SSO
  + SCIM.
- **Authorization / security docs**: Document token validation, secret encryption, and SCIM auth.

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

## Success Metrics

- Successful end-to-end login via Okta in a test Okta org (Authorization Code, org auth server).
- Okta **Run Profile** / provisioning tests pass for create, update, deactivate (or equivalent
  manual QA checklist).
- No cross-tenant credential leakage in multi-org staging tests.

## References

- [Okta: Build a Single Sign-On (SSO) integration — OpenID
  Connect](https://developer.okta.com/docs/guides/build-sso-integration/openidconnect/main/)
- [Okta: OAuth 2.0 and OpenID Connect overview][okta-oauth-overview] (supporting context)
- SCIM 2.0 ([RFC 7644](https://datatracker.ietf.org/doc/html/rfc7644)) for provisioning protocol
  semantics

[okta-oauth-overview]: https://developer.okta.com/docs/concepts/oauth-openid/
