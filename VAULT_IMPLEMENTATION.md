# Per-User Vault Management for MCP Authentication

## Overview

This implementation adds per-user JWT-based authentication for MCP (Model Context Protocol) access in the SuperPlane agent system. Previously, the system used a static bearer token. Now, each user gets their own Anthropic vault with a scoped JWT credential.

## Changes

### 1. Database Model (`pkg/models/agent_vault.go`)

Created `AgentVault` model to track user vaults:
- `id`: Primary key
- `user_id`: Foreign key to users table
- `organization_id`: Foreign key to organizations table  
- `provider_vault_id`: Anthropic's vault ID
- `provider_name`: Provider identifier (e.g., "anthropic")
- `credential_id`: Provider's credential ID within the vault
- `mcp_server_url`: URL the credential is bound to

Key functions:
- `FindOrCreateAgentVault()`: Upserts vault records
- `FindAgentVaultForUser()`: Retrieves vault for a user

### 2. Database Migration (`db/migrations/20260518214225_add-agent-vaults`)

Creates `agent_vaults` table with:
- Unique index on `(user_id, organization_id, provider_name)`
- Foreign key constraints with CASCADE delete
- Timestamps for auditing

### 3. Vault Manager (`pkg/agents/vaults.go`)

`VaultManager` handles Anthropic vault lifecycle:

**Key methods:**
- `EnsureVaultForUser()`: Returns vault ID, creating if needed
- `provisionVault()`: Creates vault + credential via Anthropic API
- `createVault()`: POST `/v1/vaults` with user metadata
- `createVaultCredential()`: POST `/v1/vaults/{id}/credentials` with JWT
- `updateVaultCredential()`: PUT to refresh JWT in existing credential

**Features:**
- Caches vault IDs in database to avoid re-creating
- Updates credentials with fresh JWTs on each session provision
- Graceful fallback if vault creation fails (logs warning, continues without vault)

### 4. Service Integration (`pkg/agents/service.go`)

Modified `Service` to include `VaultManager`:

**Changes in `provisionSession()`:**
1. Mint a user-scoped JWT before creating the session
2. Call `vaultManager.EnsureVaultForUser()` with the JWT
3. Pass `vault_id` to `CreateSession()` options
4. Vault errors are logged but don't fail session creation (fallback behavior)

**JWT scopes:**
The minted JWT includes:
- `org:read`
- `integrations:read`
- `canvases:read:<canvas_id>`
- `canvases:update_version:<canvas_id>`

These scopes authorize the MCP tools to operate on the user's canvas.

### 5. MCP Handler Updates (`pkg/mcp/handler.go`)

**Removed:**
- `staticToken` field from `Handler` struct
- Static token fallback logic in request validation

**Now:**
- Only validates JWT via `jwt.ValidateAndGetClaims()`
- Extracts `org_id` and `user_id` from JWT claims
- Uses claims for authorization (not from tool parameters)

**Constructor:**
```go
func NewHandler(jwtSigner *jwt.Signer, reg *registry.Registry) *Handler
```

### 6. Server Initialization (`pkg/server/server.go`)

Updated `buildAgentService()`:
- Creates `VaultManager` with Anthropic API key
- Reads MCP server URL from `MCP_SERVER_URL` env var (defaults to `<baseURL>/mcp`)
- Passes `vaultManager` to `NewService()`

### 7. Public Server (`pkg/public/server.go`)

Removed `MCP_STATIC_TOKEN` environment variable usage:
```go
// Before:
mcpStaticToken := os.Getenv("MCP_STATIC_TOKEN")
mcpHandler := mcp.NewHandler(s.jwt, s.registry, mcpStaticToken)

// After:
mcpHandler := mcp.NewHandler(s.jwt, s.registry)
```

## Configuration

### Environment Variables

- `MCP_SERVER_URL` (optional): URL where MCP server is accessible  
  Default: `<SUPERPLANE_URL>/mcp`

### Anthropic API Requirements

The vault manager requires:
- `ANTHROPIC_API_KEY`: API key with vault management permissions
- Anthropic API endpoint: `https://api.anthropic.com/v1` (or override via `Config.BaseURL`)

## Security Model

### JWT Minting
- Scoped JWTs minted per-canvas with 1-hour TTL
- JWTs include `user_id`, `org_id`, `purpose: "agent-builder"`
- Scopes limit access to specific canvas operations

### Vault Isolation
- One vault per `(user_id, organization_id, provider)` tuple
- Credentials bound to specific MCP server URL
- Vault metadata includes org/user IDs for auditing

### MCP Authorization
- MCP tools validate JWT signature and expiration
- Extract identity from JWT (not from tool parameters)
- Enforce scopes on all canvas/integration operations

## Error Handling

### Graceful Degradation
If vault creation fails:
1. Error logged at WARN level
2. Session creation continues without vault
3. Agent can still operate (with reduced MCP capabilities)

### Vault Update Failures
If credential update fails:
1. Logged as warning
2. System attempts to recreate vault
3. If recreation fails, continues without vault

## Migration Path

### From Static Token
1. Deploy code with database migration
2. Remove `MCP_STATIC_TOKEN` from environment
3. Existing sessions continue with static token (if still set)
4. New sessions automatically provision vaults

### Rollback
To rollback:
1. Set `MCP_STATIC_TOKEN` environment variable
2. Revert code changes (vaults remain in DB but unused)
3. Run down migration: `db/migrations/20260518214225_add-agent-vaults.down.sql`

## Testing

### Unit Tests
- `pkg/agents/service_test.go`: Updated to pass `nil` vault manager (tests pass)
- `pkg/mcp/handler_test.go`: Already compatible with new signature

### Integration Testing
Test scenarios:
1. First session creation → vault provisioned
2. Second session for same user → vault reused
3. JWT expiration → credential updated
4. Vault creation failure → session still created
5. MCP tool call with valid JWT → authorized
6. MCP tool call with invalid JWT → rejected

## Performance Considerations

### Vault Caching
- Vault IDs cached in database
- Reduces Anthropic API calls to 1 per user (first session)
- Subsequent sessions only update credential (1 API call)

### JWT Refresh
- JWTs minted on every session provision
- 1-hour TTL balances security and performance
- Credential updates are fast PUT requests

## Future Enhancements

1. **Credential Rotation**: Periodic JWT refresh for long-running sessions
2. **Vault Cleanup**: Delete unused vaults after X days of inactivity
3. **Multi-Provider**: Extend to support non-Anthropic providers
4. **Audit Logging**: Track vault access patterns
5. **Quota Management**: Enforce per-user vault limits

## API Endpoints Used

### Anthropic Vaults API
- `POST /v1/vaults`: Create vault
  ```json
  {
    "display_name": "SuperPlane MCP (user-id)",
    "metadata": {
      "organization_id": "...",
      "user_id": "...",
      "purpose": "mcp-auth"
    }
  }
  ```

- `POST /v1/vaults/{id}/credentials`: Create credential
  ```json
  {
    "auth": {
      "type": "static_bearer",
      "mcp_server_url": "https://...",
      "token": "<jwt>"
    }
  }
  ```

- `PUT /v1/vaults/{id}/credentials/{cred_id}`: Update credential (same body as POST)

## References

- JWT implementation: `pkg/jwt/scoped.go`
- Agent session lifecycle: `pkg/agents/service.go`
- MCP protocol handler: `pkg/mcp/handler.go`
- Anthropic provider: `pkg/agents/anthropic/provider.go`
