# Service Accounts

## Overview

Service accounts are non-human identities that provide programmatic access to the SuperPlane API. They enable CI/CD pipelines, automation scripts, external integrations, and machine-to-machine communication without being tied to any individual user's credentials.

Today, the only way to access the SuperPlane API programmatically is through personal user API tokens. This creates several problems:

- **Lifecycle coupling**: When a user leaves the organization, their token is revoked, breaking any automation that relied on it.
- **Shared credentials**: Teams often share a single user's token across multiple services, making it impossible to trace which system performed an action.
- **Over-permissioning**: Personal tokens inherit the user's full role, even when the automation only needs a narrow set of permissions.
- **No rotation story**: There is no way to rotate tokens without downtime, since each user only has a single token.

Service accounts solve these problems by introducing a first-class, non-human identity that is managed independently of any individual user.

## Goals

1. Allow organizations to create, manage, and delete service accounts.
2. Support multiple API tokens per service account, enabling zero-downtime rotation.
3. Integrate with the existing RBAC system so service accounts can be assigned roles and added to groups, just like regular users.
4. Provide a clear audit trail — every API action performed by a service account should be attributable to that specific service account.
5. Expose full management capabilities through the API and the web UI.

## Non-Goals

- **OAuth2 client credentials flow**: Service accounts authenticate via API tokens, not OAuth2. An OAuth2 integration may be built on top of this in the future.
- **Cross-organization service accounts**: Each service account is scoped to a single organization, matching the existing tenant isolation model.
- **Fine-grained per-token permissions**: All tokens belonging to a service account share the same permissions. Per-token scoping is out of scope for the initial implementation.
- **Automatic token rotation**: Users manage rotation manually via the API/UI. Automated rotation policies can be added later.

## Detailed Design

### Data Model

A service account is a new entity scoped to an organization. It sits alongside users in the authorization system but is clearly distinguishable as non-human.

#### `service_accounts` Table

| Column            | Type        | Description                                      |
|-------------------|-------------|--------------------------------------------------|
| `id`              | `uuid`      | Primary key.                                     |
| `organization_id` | `uuid`      | FK to `organizations`. Enforces tenant isolation. |
| `name`            | `string`    | Human-readable name. Unique within the org.       |
| `description`     | `text`      | Optional description of what this account is for. |
| `created_by`      | `uuid`      | FK to `users`. The user who created this account. |
| `created_at`      | `timestamp` | Creation time.                                   |
| `updated_at`      | `timestamp` | Last update time.                                |

#### `service_account_tokens` Table

Each service account can have multiple tokens. This is the key enabler for zero-downtime rotation: create a new token, update consumers, then delete the old token.

| Column               | Type        | Description                                          |
|----------------------|-------------|------------------------------------------------------|
| `id`                 | `uuid`      | Primary key.                                         |
| `service_account_id` | `uuid`      | FK to `service_accounts`.                             |
| `name`               | `string`    | Human-readable label (e.g., "CI pipeline token").     |
| `token_hash`         | `string`    | SHA-256 hash of the token. The raw token is never stored. |
| `last_used_at`       | `timestamp` | Last time this token was used for authentication. Nullable. |
| `expires_at`         | `timestamp` | Optional expiration time. Null means no expiration.   |
| `created_at`         | `timestamp` | Creation time.                                       |

### Authentication

Service account tokens authenticate exactly like existing user API tokens — via the `Authorization: Bearer <token>` header. The authentication middleware needs to be extended to look up tokens from both the `users` table and the `service_account_tokens` table.

**Token lookup flow:**

1. Extract the Bearer token from the `Authorization` header.
2. Hash the token with SHA-256.
3. Look up the hash in `users.token_hash` (existing behavior).
4. If not found, look up the hash in `service_account_tokens.token_hash`.
5. If found, load the associated `service_account` and verify it belongs to an active organization.
6. Check `expires_at` — reject the request if the token has expired.
7. Update `last_used_at` asynchronously (fire-and-forget or batched).
8. Set request context with the service account identity (instead of a user identity).

The authorization interceptor currently expects `x-user-id` and `x-organization-id` in the gRPC metadata. For service accounts, the middleware will set:

- `x-user-id` → service account ID (the RBAC system already treats this as opaque; it works with any string identifier).
- `x-organization-id` → the service account's organization ID.

This means the existing Casbin permission checks work with zero changes — a service account's ID is used as the "user" in policy evaluation, and roles/groups are assigned to this ID.

### Authorization

Service accounts participate in the existing RBAC system as first-class principals:

- **Role assignment**: An org admin can assign any role to a service account (e.g., `org_viewer`, `org_admin`, or a custom role).
- **Group membership**: A service account can be added to groups, inheriting the group's role.
- **Permission enforcement**: The authorization interceptor does not need changes. It checks permissions based on the user ID in the context, which will be the service account's ID for service account requests.

**Restriction**: Service accounts cannot be assigned the `org_owner` role. Ownership is reserved for human users.

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

  rpc CreateServiceAccountToken(CreateServiceAccountTokenRequest) returns (CreateServiceAccountTokenResponse);
  rpc ListServiceAccountTokens(ListServiceAccountTokensRequest) returns (ListServiceAccountTokensResponse);
  rpc DeleteServiceAccountToken(DeleteServiceAccountTokenRequest) returns (DeleteServiceAccountTokenResponse);
}
```

#### REST Endpoints (via gRPC-Gateway)

| Method   | Path                                                         | Description                        |
|----------|--------------------------------------------------------------|------------------------------------|
| `POST`   | `/api/v1/service-accounts`                                   | Create a service account.          |
| `GET`    | `/api/v1/service-accounts`                                   | List service accounts in the org.  |
| `GET`    | `/api/v1/service-accounts/{id}`                              | Get service account details.       |
| `PATCH`  | `/api/v1/service-accounts/{id}`                              | Update name/description.           |
| `DELETE` | `/api/v1/service-accounts/{id}`                              | Delete a service account.          |
| `POST`   | `/api/v1/service-accounts/{id}/tokens`                       | Create a new token.                |
| `GET`    | `/api/v1/service-accounts/{id}/tokens`                       | List tokens (metadata only).       |
| `DELETE` | `/api/v1/service-accounts/{id}/tokens/{token_id}`            | Delete a specific token.           |

#### Authorization Rules

| Endpoint                     | Required Permission          |
|------------------------------|------------------------------|
| `CreateServiceAccount`       | `service_accounts:create`    |
| `ListServiceAccounts`        | `service_accounts:read`      |
| `DescribeServiceAccount`     | `service_accounts:read`      |
| `UpdateServiceAccount`       | `service_accounts:update`    |
| `DeleteServiceAccount`       | `service_accounts:delete`    |
| `CreateServiceAccountToken`  | `service_accounts:update`    |
| `ListServiceAccountTokens`   | `service_accounts:read`      |
| `DeleteServiceAccountToken`  | `service_accounts:update`    |

These permissions should be added to the `org_admin` and `org_owner` roles. The `org_viewer` role gets `service_accounts:read` only.

### Token Generation

Tokens are generated as cryptographically random strings with a recognizable prefix:

- Format: `sp_sa_<random-32-bytes-hex>` (total ~70 characters).
- The `sp_sa_` prefix makes it easy to identify service account tokens in logs and secret scanners.
- The raw token is returned **only once** at creation time. After that, only the hash is stored.
- Hashing uses the same `crypto.HashToken` function already used for user API tokens.

### Web UI

The service accounts management UI should be accessible under **Organization Settings > Service Accounts**.

#### List View

- Table showing: name, description, role, number of tokens, creation date, created by.
- Actions: create new, view details, delete.

#### Detail View

- Service account metadata (name, description, role).
- Edit name and description inline.
- Tokens section:
  - Table of tokens: name, created date, last used date, expiration, and a delete button.
  - "Create Token" button that shows the raw token once with a copy button and a warning that it won't be shown again.
- Role assignment section:
  - Current role displayed with option to change.
- Group membership section:
  - List of groups the service account belongs to with option to add/remove.

## Implementation Plan

### Phase 1: Core Backend

1. Create database migration for `service_accounts` and `service_account_tokens` tables.
2. Create Go models in `pkg/models/` (`service_account.go`, `service_account_token.go`).
3. Define protobuf service in `protos/service_accounts.proto`.
4. Implement gRPC actions in `pkg/grpc/actions/service_accounts/`.
5. Add RBAC permissions (`service_accounts:create/read/update/delete`) to the organization policy templates.
6. Add authorization rules to the interceptor in `pkg/authorization/interceptor.go`.
7. Extend the authentication middleware in `pkg/public/middleware/auth.go` to resolve service account tokens.

### Phase 2: API Integration

1. Register gRPC-Gateway routes in `pkg/public/server.go`.
2. Regenerate protobuf, OpenAPI spec, and SDK clients.
3. Add E2E tests for all service account CRUD operations and token authentication.

### Phase 3: Web UI

1. Add "Service Accounts" page under Organization Settings.
2. Implement list view with create/delete actions.
3. Implement detail view with token management.
4. Implement role and group management for service accounts.

## Security Considerations

- **Token storage**: Raw tokens are never stored. Only SHA-256 hashes are persisted.
- **Token display**: The raw token is shown exactly once at creation time. It cannot be retrieved afterwards.
- **Token expiration**: Tokens can optionally have an expiration date. Expired tokens are rejected at authentication time.
- **Deletion cascade**: Deleting a service account deletes all its tokens and removes all RBAC policies associated with it.
- **No owner role**: Service accounts cannot be assigned the `org_owner` role to prevent privilege escalation through non-human identities.
- **Rate limiting**: Service account token authentication should follow the same rate-limiting rules as user token authentication.
- **Secret scanning**: The `sp_sa_` token prefix enables integration with secret scanning tools (e.g., GitHub secret scanning).

## Decisions

- **Maximum tokens per service account**: 10 tokens per service account.
- **Service account quotas**: 100 service accounts per organization.
- **Token activity log**: Only `last_used_at` is tracked per token. No detailed usage history.
- **Impersonation**: Not supported. Service accounts cannot be impersonated through the UI.
