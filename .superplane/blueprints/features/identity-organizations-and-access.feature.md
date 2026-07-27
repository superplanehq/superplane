# Identity, Organizations, and Access Feature Blueprint

## Feature Summary

People authenticate once, participate in one or more organizations as users,
invite/remove members, and administer roles, groups, API keys, and scoped
permissions. This feature implements the
[Identity and Access](../../requirements/identity-and-access.md) and
[Organization Administration](../../requirements/organization-administration.md)
requirements, with uncovered behavior identified below.

Evidence: [`pkg/authentication/`](../../../pkg/authentication/),
[`pkg/authorization/`](../../../pkg/authorization/),
[`protos/users.proto`](../../../protos/users.proto),
[`protos/groups.proto`](../../../protos/groups.proto), and
[`protos/roles.proto`](../../../protos/roles.proto).

## Component Blueprint Composition

- [Authentication and RBAC](../components/authentication-and-rbac.component.md):
  #AuthenticationHandler establishes account sessions and
  #AuthenticationRBAC resolves organization/domain permissions.
- [API Gateway and Realtime](../components/api-gateway-and-realtime.component.md):
  native login routes and generated organization/user/group/role/API-key
  services share the public HTTP boundary.
- [PostgreSQL](../containers/postgresql.container.md): stores `Account`,
  `AccountProvider`, `User`, `Organization`, group/role metadata, invitations,
  invite links, API keys, password hashes, and magic-code records.

## Feature-Specific Flow

Configured login routes authenticate or create an `Account`, issue a signed
session, and resolve a `User` in the selected organization. Administrators use
organization services to invite/remove members and assign role/group policy.
Every protected request maps to a declarative `AuthorizationRule`; scoped API
principals additionally bind permission to a resource ID.

## System Contracts

- Email/account identity and organization membership are separate records.
- A user cannot use membership in one organization to address another
  organization’s resources.
- Signup, password login, magic code, OAuth providers, and owner setup are
  installation-configured surfaces.
- Magic codes expire, are rate limited, and cap verification attempts.
- JWT/OIDC/session signing and encryption material are deployment secrets.
- Experimental feature gates supplement, but do not replace, permissions.
- New protected API paths require an authorization rule before release.

## Requirement Coverage

- **REQ-IAM-001:** Session authentication, protected routes, and
  organization-scoped users provide authenticated organization entry.
- **REQ-IAM-002:** Owner-setup routes and installation-owner authorization
  provide first-run bootstrap.
- **REQ-IAM-003:** Organization API-key services create, identify, regenerate,
  authenticate, and delete programmatic credentials.
- **REQ-IAM-004:** Declarative authorization rules and scoped checks enforce the
  same permissions at the API boundary that the UI uses to gate actions.
- **REQ-IAM-005:** Gap: the current identity model has users and API keys, but
  no organization service-identity model or API with assignable roles.
- **REQ-ORG-001:** Organization services and settings persist tenant lifecycle
  changes and expose them through organization selection.
- **REQ-ORG-002:** Invitation links and membership services implement joining,
  listing, and removal.
- **REQ-ORG-003:** Group membership, role assignment, and authorization
  resolution implement reusable access policy.
- **REQ-ORG-004:** Installation administration routes and domain-level
  authorization guard organization, account, usage, and task administration.

## Architecture Decision Records

### ADR-001: Separate global accounts from tenant users

**Context:** The same person may have different roles in multiple
organizations.

**Decision:** Represent identity as global `Account` and membership as
organization-scoped `User`.

**Consequences:** Authentication is reusable while authorization always needs
an explicit organization context.

## Open Questions

- What session revocation behavior is required after password, role, or
  membership changes?
- Should all enabled login methods expose uniform audit events?
