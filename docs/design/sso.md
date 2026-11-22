# Organization Single Sign-On

SuperPlane can connect one OpenID Connect identity provider to an organization.
Members sign in through that provider. The provider creates and deactivates
members through SCIM 2.0 Users.

The protocol sits behind an adapter. A later change can add SAML 2.0 without a
change to the sign-in pipeline. This release does not include SAML, SCIM
Groups, or Microsoft Entra ID tests. The Entra preset sets the subject claim
to `oid`. That preset is not tested.

## Decisions

- The organization owns the connection. The account stays global.
- Identities are stored in `sso_identities`.
- The identity provider is found from a verified email domain. One verified
  domain belongs to one organization.
- Deactivation removes the member from the organization. It does not block the
  global account.
- Enforcement runs on each cookie request.
- The license feature `sso` is required only to create or expand access.
  Existing connections continue to work when the grant is absent.
- Login state is a signed cookie. There is no login-state table.

## Data

- `organization_domains` stores a domain and its verification token.
- `sso_connections` stores one connection per organization. The client secret
  is encrypted.
- `sso_identities` links a subject to an account in that organization. The
  ID token is encrypted and is stored only when identity-provider sign-out is
  on.
- `scim_tokens` stores the hash of a bearer token. A connection can have five
  active tokens.
- `users.provisioned_by` is `jit` or `scim` when Single Sign-On created the
  member. It is empty for other members.

## Sign-in

1. The browser opens `/auth/sso/{connection_id}/start`.
2. SuperPlane stores state, nonce, and the PKCE verifier in a cookie for 10
   minutes.
3. The identity provider returns an authorization code.
4. SuperPlane verifies the ID token and links the account.
5. SuperPlane creates a member when just-in-time access is on and the license
   and a seat are available.
6. The session cookie stores `connection_id:auth_unix` in the `sso` claim.

SuperPlane links an existing account by email only when this organization has
verified the email domain. A blocked account cannot sign in.

## Enforcement

An active connection with enforcement `required` blocks cookie access to that
organization until the session has a fresh SSO entry. Owner bypass is optional.
Bearer tokens and support sessions are exempt. The block does not disable
other sign-in methods for the account.

## SCIM

The base URL is `/scim/v2/{connection_id}`. Only the User resource is
supported. `/Groups` returns 404. Deactivation uses the same member removal
rules as the organization settings page. Deactivation of the last owner keeps
the membership and marks the identity inactive.

## Known limitations

- Personal API tokens of a removed member are deleted. Organization API keys
  that the member created stay valid.
- A change to the email at the identity provider does not change
  `accounts.email`.
- SCIM Groups and group-to-role mapping are not included.
- SAML 2.0 is not included.
- Microsoft Entra ID is not tested.
