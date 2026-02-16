# Service Accounts

## Overview

Service accounts are non-human identities that provide programmatic access to the SuperPlane API. They enable CI/CD pipelines, automation scripts, external integrations, and machine-to-machine communication without being tied to any individual user's credentials.

Today, the only way to access the SuperPlane API programmatically is through personal user API tokens. This creates several problems:

- **Lifecycle coupling**: When a user leaves the organization, their token is revoked, breaking any automation that relied on it.
- **Shared credentials**: Teams often share a single user's token across multiple services, making it impossible to trace which system performed an action.
- **Over-permissioning**: Personal tokens inherit the user's full role, even when the automation only needs a narrow set of permissions.

Service accounts solve these problems by introducing a first-class, non-human identity that is managed independently of any individual user.

## Goals

1. Allow organizations to create, manage, and delete service accounts.
2. Provide API token authentication for service accounts, reusing the existing token mechanism.
3. Integrate with the existing RBAC system so service accounts can be assigned roles and added to groups, just like regular users.
4. Provide a clear audit trail — every API action performed by a service account should be attributable to that specific service account.
5. Expose full management capabilities through the API and the web UI.

## Non-Goals

- **OAuth2 client credentials flow**: Service accounts authenticate via API tokens, not OAuth2. An OAuth2 integration may be built on top of this in the future.
- **Cross-organization service accounts**: Each service account is scoped to a single organization, matching the existing tenant isolation model.
- **Multiple tokens per service account**: The initial implementation uses a single token per service account (reusing the existing `token_hash` mechanism). Multi-token support for zero-downtime rotation can be added later.
- **Token expiration**: Tokens do not expire in the initial implementation. Expiration support can be added with multi-token support.

## Architecture Decision: Unified vs. Separate Tables

### Options Considered

**Option A — Separate `service_accounts` table:** Service accounts live in their own table, completely independent of `users`. Both participate in the same Casbin RBAC system as interchangeable principals. Every code path that receives an identity from the request context needs a polymorphic helper to resolve the ID from either table.

**Option B — Unified `users` table with a `type` column:** Service accounts are rows in the existing `users` table with `type = 'service_account'` and a nullable `account_id`. All existing code that looks up users by ID works unchanged. Only the few places that manage user-type-specific behavior (invitations, profile page, account providers) need guards.

### Decision: Option B (Unified Table)

We chose the unified table approach for the following reasons:

**Developer experience.** With separate tables, every developer writing code that consumes an identity must remember to use a polymorphic helper instead of the direct `FindActiveUserByID` call. If they forget, it works in tests (which use human users) and only breaks when a customer uses a service account — a subtle, hard-to-catch bug class. With a unified table, `FindActiveUserByID` just works for both.

**Complexity distribution.** Option A distributes complexity across every consumer of identity (any endpoint, any serialization path, any component action). Option B concentrates complexity in a few identity-management spots (creation, profile, invitations). There are far more consumers than producers.

**Codebase impact analysis.** We audited every call site that would be affected. With Option A, 14+ call sites across auth middleware, gRPC actions, serialization, and component contexts need changes. With Option B, only 3-4 spots that access `AccountID` or user-type-specific features need guards.

**Industry context.** Some platforms (GCP, Kubernetes, Azure, GitHub) use separate entities. Others (AWS IAM, GitLab, Grafana) use a unified model. Both approaches are proven in production.

### Trade-offs Accepted

- `account_id` becomes nullable on the `users` table. Service accounts have no account.
- The `unique_user_in_organization` constraint `(organization_id, account_id, email)` needs adjustment — service accounts don't have an `account_id` or a meaningful email.
- ~3-4 places that access `user.AccountID` need nil checks (primarily `convertUserToProto` in `pkg/grpc/actions/auth/common.go` and invitation flows).
- The `/api/v1/me` endpoints need to handle service account callers (either return a service-account-specific response or block them).

## Detailed Design

### Data Model

Service accounts are stored in the existing `users` table with a `type` column to distinguish them from human users. Service accounts reuse the existing `token_hash` column for API token authentication — no new tables are needed.

#### Changes to `users` Table

| Change                | Details                                                         |
|-----------------------|-----------------------------------------------------------------|
| Add `type` column     | `string`, default `'human'`. Values: `'human'`, `'service_account'`. |
| Add `description` column | `text`, nullable. Used by service accounts for description.  |
| Make `account_id` nullable | Service accounts have no account. Null for `type = 'service_account'`. |
| Make `email` nullable | Service accounts have no email. Null for `type = 'service_account'`. |
| Add `created_by` column | `uuid`, nullable. FK to `users`. Records who created the service account. Null for human users. |
| Adjust unique constraint | Replace `unique_user_in_organization(organization_id, account_id, email)` with two partial unique indexes: one for human users on `(organization_id, account_id, email) WHERE type = 'human'` and one for service accounts on `(organization_id, name) WHERE type = 'service_account'`. |

No new tables are required. Service accounts use the existing `token_hash` column on the `users` table, which is the same mechanism human users already use for API tokens.

### Authentication

Service account tokens authenticate exactly like existing user API tokens — via the `Authorization: Bearer <token>` header. Because service accounts are rows in the `users` table and use the same `token_hash` column, **no changes to the authentication middleware are required.**

The existing flow already works:

1. Extract the Bearer token from the `Authorization` header.
2. Hash the token with SHA-256.
3. Look up the hash in `users.token_hash` via `FindActiveUserByTokenHash`.
4. Set request context with the `*models.User`.

Since service accounts are `users` rows, step 3 matches them automatically. The gateway handler (`grpcGatewayHandler`) and authorization interceptor work without modification. The `x-user-id` and `x-organization-id` metadata is set from `user.ID` and `user.OrganizationID` as usual.

### Authorization

Service accounts participate in the existing RBAC system as first-class principals:

- **Role assignment**: An org admin can assign any role to a service account (e.g., `org_viewer`, `org_admin`, or a custom role).
- **Group membership**: A service account can be added to groups, inheriting the group's role.
- **Permission enforcement**: The authorization interceptor does not need changes. It checks permissions based on the user ID in the context, which is the service account's user ID.

**Restriction**: Service accounts cannot be assigned the `org_owner` role. Ownership is reserved for human users.

### Codebase Impact

The following areas require changes to support service accounts. Because we chose the unified table approach, most existing code works unchanged.

#### No Changes Required

These work automatically because service accounts are rows in the `users` table:

- **Auth middleware** (`pkg/public/middleware/auth.go`) — `FindActiveUserByTokenHash` already matches service account tokens since they use the same `token_hash` column.
- **Authorization interceptor** (`pkg/authorization/interceptor.go`) — already uses opaque user ID strings.
- **Casbin RBAC** (`pkg/authorization/service.go`) — role assignment, permission checks, group membership.
- **Gateway handler** (`pkg/public/server.go` `grpcGatewayHandler`) — reads `user.ID` and `user.OrganizationID`.
- **`AuthContext`** (`pkg/workers/contexts/auth_context.go`) — holds `*models.User`, works for both types.
- **Component runtime** (approval, wait, time gate) — uses `core.User{ID, Name, Email}` from `AuthContext`.
- **Canvas/Blueprint `created_by`** — stores a user UUID, serialization resolves it via `FindMaybeDeletedUserByID`, which works.
- **Execution `cancelled_by`** — stores a user UUID, batch-resolved via `FindMaybeDeletedUsersByIDs`, which works.
- **Secret `created_by`** — stores the user ID string directly, works.
- **Trigger/execution actions** (`InvokeNodeTriggerAction`, `InvokeNodeExecutionAction`, `CancelExecution`) — call `FindActiveUserByID`, which works.

#### Changes Required

| Area | File(s) | Change |
|------|---------|--------|
| **User serialization** | `pkg/grpc/actions/auth/common.go` | `convertUserToProto` calls `FindAccountByID(user.AccountID)` — needs nil check for service accounts (no account). Return a simplified proto with no account providers. |
| **User listing** | `pkg/grpc/actions/auth/common.go` | `GetUsersWithRolesInDomain` / `ListUsers` — decide whether service accounts appear in the user list or need a separate listing. |
| **`/api/v1/me`** | `pkg/grpc/actions/me/get_user.go` | Return a response that works for both types. |
| **`/api/v1/me/token`** | `pkg/grpc/actions/me/regenerate_token.go` | Block for service accounts — they manage tokens via the service account token endpoints. |
| **Invitation flow** | `pkg/grpc/actions/organizations/create_invitation.go` | No change needed — invitations work by email, and service accounts have no email. Naturally excluded. |
| **Assign role** | `pkg/grpc/actions/auth/assign_role.go` | `FindUser` resolves by ID or email. Works for service accounts (by ID). Add guard to prevent `org_owner` assignment. |
| **Delete organization** | `pkg/grpc/actions/organizations/delete_organization.go` | No change needed — uses user ID for logging only. |

### API Design

All service account endpoints are organization-scoped and follow the existing API patterns.

#### gRPC Service Definition

```protobuf
service ServiceAccounts {
  rpc CreateServiceAccount(CreateServiceAccountRequest) returns (CreateServiceAccountResponse);
  rpc ListServiceAccounts(ListServiceAccountsRequest) returns (ListServiceAccountsResponse);
  rpc DescribeServiceAccount(DescribeServiceAccountRequest) returns (DescribeServiceAccountResponse);
  rpc UpdateServiceAccount(UpdateServiceAccountRequest) returns (UpdateServiceAccountResponse);
  rpc DeleteServiceAccount(DeleteServiceAccountRequest) returns (DeleteServiceAccountResponse);
  rpc RegenerateServiceAccountToken(RegenerateServiceAccountTokenRequest) returns (RegenerateServiceAccountTokenResponse);
}
```

#### REST Endpoints (via gRPC-Gateway)

| Method   | Path                                                         | Description                              |
|----------|--------------------------------------------------------------|------------------------------------------|
| `POST`   | `/api/v1/service-accounts`                                   | Create a service account.                |
| `GET`    | `/api/v1/service-accounts`                                   | List service accounts in the org.        |
| `GET`    | `/api/v1/service-accounts/{id}`                              | Get service account details.             |
| `PATCH`  | `/api/v1/service-accounts/{id}`                              | Update name/description.                 |
| `DELETE` | `/api/v1/service-accounts/{id}`                              | Delete a service account.                |
| `POST`   | `/api/v1/service-accounts/{id}/token`                        | Regenerate the service account's token.  |

#### Authorization Rules

| Endpoint                          | Required Permission          |
|-----------------------------------|------------------------------|
| `CreateServiceAccount`            | `service_accounts:create`    |
| `ListServiceAccounts`             | `service_accounts:read`      |
| `DescribeServiceAccount`          | `service_accounts:read`      |
| `UpdateServiceAccount`            | `service_accounts:update`    |
| `DeleteServiceAccount`            | `service_accounts:delete`    |
| `RegenerateServiceAccountToken`   | `service_accounts:update`    |

These permissions should be added to the `org_admin` and `org_owner` roles. The `org_viewer` role gets `service_accounts:read` only.

### Token Management

Service accounts use the same single-token mechanism as human users (the `token_hash` column on the `users` table). Token management works identically to the existing `RegenerateToken` endpoint for human users:

- A token is generated as a cryptographically random string using the existing `crypto.Base64String` function.
- The hash is stored in `users.token_hash` using the existing `crypto.HashToken` function.
- The raw token is returned **only once** at creation time. After that, only the hash is stored.
- Regenerating a token replaces the previous one immediately.

**Future enhancement**: Multi-token support (via a dedicated `service_account_tokens` table) can be added later to enable zero-downtime rotation, token expiration, named tokens, and per-token usage tracking.

### Web UI

The service accounts management UI should be accessible under **Organization Settings > Service Accounts**.

#### List View

- Table showing: name, description, role, has token, creation date, created by.
- Actions: create new, view details, delete.

#### Detail View

- Service account metadata (name, description, role).
- Edit name and description inline.
- Token section:
  - Shows whether a token exists.
  - "Regenerate Token" button that shows the raw token once with a copy button and a warning that it won't be shown again.
- Role assignment section:
  - Current role displayed with option to change.
- Group membership section:
  - List of groups the service account belongs to with option to add/remove.

## Implementation Plan

### Phase 1: Schema Migration

1. Add `type` column (`string`, default `'human'`) to `users` table.
2. Add `description` column (`text`, nullable) to `users` table.
3. Add `created_by` column (`uuid`, nullable) to `users` table.
4. Make `account_id` nullable on `users` table.
5. Make `email` nullable on `users` table.
6. Replace `unique_user_in_organization` with partial unique indexes.
7. Run migration against dev and test databases.

### Phase 2: Core Backend

1. Update `models.User` struct to include `Type`, `Description`, and `CreatedBy` fields.
2. Add `UserTypeHuman` and `UserTypeServiceAccount` constants to `pkg/models/constants.go`.
3. Update `convertUserToProto` in `pkg/grpc/actions/auth/common.go` to handle nullable `AccountID`.
4. Define protobuf service in `protos/service_accounts.proto`.
5. Implement gRPC actions in `pkg/grpc/actions/service_accounts/`.
6. Add RBAC permissions (`service_accounts:create/read/update/delete`) to the organization policy templates.
7. Add authorization rules to the interceptor in `pkg/authorization/interceptor.go`.
8. Guard `/api/v1/me/token` endpoint against service account callers.

### Phase 3: API Integration

1. Register gRPC-Gateway routes in `pkg/public/server.go`.
2. Regenerate protobuf, OpenAPI spec, and SDK clients.
3. Add E2E tests for all service account CRUD operations and token authentication.

### Phase 4: Web UI

1. Add "Service Accounts" page under Organization Settings.
2. Implement list view with create/delete actions.
3. Implement detail view with token management.
4. Implement role and group management for service accounts.

## Security Considerations

- **Token storage**: Raw tokens are never stored. Only SHA-256 hashes are persisted (same as human user tokens).
- **Token display**: The raw token is shown exactly once at creation/regeneration time. It cannot be retrieved afterwards.
- **Deletion cascade**: Deleting a service account clears its token hash and removes all RBAC policies associated with it.
- **No owner role**: Service accounts cannot be assigned the `org_owner` role to prevent privilege escalation through non-human identities.
- **Rate limiting**: Service account token authentication follows the same rate-limiting rules as user token authentication (same code path).

## Decisions

- **Data model**: Unified `users` table with a `type` column (see Architecture Decision section above).
- **Token model**: Single token per service account, reusing the existing `token_hash` column. Multi-token support is a future enhancement.
- **Service account quotas**: 100 service accounts per organization.
- **Impersonation**: Not supported. Service accounts cannot be impersonated through the UI.
