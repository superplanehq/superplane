# Authentication and RBAC Component Blueprint

## Capability Summary

Authentication establishes an `Account` session or API principal. RBAC resolves
that identity to an organization-scoped `User` and authorizes a resource/action
pair before domain logic. The capability covers browser sessions, bearer/API
tokens, configurable password and magic-code login, optional OAuth/OIDC
boundaries, roles, groups, and scoped tokens.

Evidence: [`pkg/authentication/`](../../../pkg/authentication/),
[`pkg/authorization/`](../../../pkg/authorization/),
[`pkg/public/server.go`](../../../pkg/public/server.go), and
[`protos/roles.proto`](../../../protos/roles.proto).

## Core Components

```component
name: AuthenticationHandler
container: API and Web
responsibilities:
  - Establishing `Account` sessions and issuing signed JWT cookies
  - Handling configured login, signup, logout, and provider callbacks
```

```component
name: AuthenticationRBAC
container: API and Web / Workers
responsibilities:
  - Resolving principals and organization/domain context
  - Enforcing `AuthorizationRule` resource/action permissions
```

```model
name: OrganizationMembership
store: PostgreSQL
description: `Account` identity represented as one `User` per organization.
fields:
  - account_id: UUID
  - organization_id: UUID
constraints:
  - Permissions are evaluated within the resolved organization/domain.
```

#PublicHTTPServer depends on #AuthenticationHandler for session establishment
and on #AuthenticationRBAC for protected routes. Gateway middleware maps exact
HTTP method/path templates to `AuthorizationRule`; services also use the
authorization abstraction for actions such as managed-agent tools.

## System Contracts

### Key Contracts

- Authentication and authorization are distinct: a valid account session does
  not imply access to an organization resource.
- Route rules declare `Resource`, `Action`, `DomainType`, optional resource path
  parameters, legacy actions, and required experimental features.
- Canvas-scoped tokens are constrained with IDs extracted from configured path
  parameters.
- Managed-agent routes require both agent permission and the corresponding
  organization experimental feature.
- Provider access/refresh tokens and other sensitive values are encrypted before
  persistence when encryption is enabled.
- Password and magic-code routes do not exist when their feature flags are off;
  signup blocking is installation configuration.

### Integration Contracts

- Identity providers and OIDC consumers are external trust boundaries.
- `DefaultAuthorizationRules()` must be updated when a new protected API route
  is added.
- Role/group CRUD is exposed through proto services and stored in PostgreSQL;
  policy reload is enabled in the API deployment.

## Architecture Decision Records

### ADR-001: Scope membership and authorization by organization

**Context:** One account can participate in multiple tenants with different
permissions.

**Decision:** Model identity as `Account` plus organization-specific `User` and
evaluate declarative resource/action policies in that domain.

**Consequences:** Tenant context is explicit and permissions can differ by
organization. Every resource lookup and route rule must preserve that scope.

## Open Questions

- Which native HTTP routes should be represented in the same rule map as
  proto-backed gateway routes?
- What policy-reload consistency is expected across API and worker replicas?
