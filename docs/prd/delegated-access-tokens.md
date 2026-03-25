# Scoped Tokens

## Overview

This PRD defines the scoped token mechanism for SuperPlane.

The app mints a short-lived JWT for an existing organization user. That JWT is narrower than the
user's full RBAC grants and is intended for one concrete use case, such as AI agent chat.

The first consumer is the AI canvas builder. The same mechanism must also support future background
agents and internal automation without introducing a second auth model.

## Problem Statement

The current agent auth model has two gaps:

- The browser can call the agent directly without first going through standard SuperPlane auth.
- The agent calls the SuperPlane API using a long-lived API token exposed to the agent runtime.

This creates an avoidable trust boundary outside the app, weakens least-privilege, and makes it
hard to support future background agents with scoped access.

SuperPlane already has organization-scoped users, service accounts, API token auth, and RBAC.
What is missing is a way to mint a temporary token for a specific user and narrow what that token
may do for a single run.

## Goals

1. Allow the app to mint short-lived JWTs for an existing `users.id` principal.
2. Allow each minted token to carry narrower capabilities than the principal's full RBAC grants.
3. Reuse the current JWT secret for v1.
4. Keep the mechanism generic so it can power agent chat, background agents, and future automation.
5. Preserve existing RBAC as a required enforcement layer for every API call.
6. Support both human users and service accounts without separate token formats.
7. Provide a clear implementation path for the current AI canvas builder.

## Non-Goals

- Replacing the existing browser session cookie flow.
- Replacing long-lived user or service account API tokens for existing CLI or automation clients.
- Introducing OAuth2 or external token exchange flows.
- Defining every future scoped-token consumer in this document.
- Building token revocation or token introspection in v1.
- Changing the current JWT secret management model in v1.

## Users / Personas

- Organization members using the AI canvas builder in the web UI.
- Organization admins who will later run background agents through service accounts.
- Backend engineers implementing new internal automation that needs scoped temporary access.
- Security-conscious operators who need least-privilege and clear audit semantics.

## Success Metrics

1. The AI canvas builder no longer requires `SUPERPLANE_API_TOKEN` in the agent runtime.
2. The scoped token model supports both a human user subject and a service account subject.
3. Every scoped token has a maximum lifetime of 15 minutes or less in v1.
4. Every scoped API call must pass both scoped-token checks and existing RBAC checks.
5. The AI canvas builder no longer depends on unauthenticated browser-to-agent access.
6. Failed scoped-token validation returns `401`, and denied scoped permissions continue to follow
   the existing not-found masking behavior where applicable.

## Product Requirements

### Principal Model

- The scoped token subject must be a `users.id`, not an `accounts.id`.
- The same token format must work for both human users and service accounts.
- The API must resolve the subject from the `users` table after validating the JWT.
- The API must verify that the subject belongs to the `org_id` carried by the token.

### Token Shape

- Scoped tokens must be JWTs signed with the current SuperPlane JWT secret in v1.
- Scoped tokens must include `token_type=scoped`.
- Scoped tokens must include `aud=superplane_api`.
- Scoped tokens must include `sub=<users.id>`.
- Scoped tokens must include `org_id=<organizations.id>`.
- Scoped tokens must include `purpose=<use case name>`, such as `agent_chat`.
- Scoped tokens must include `permissions`, as a non-empty list.
- Each permission must contain:
  - `resourceType`
  - `action`
  - optional `resources`
- `resources` must be used only when the token needs object-level narrowing.
- Scoped tokens must include `iat`, `nbf`, and `exp`.

Example:

```json
{
  "token_type": "scoped",
  "aud": "superplane_api",
  "sub": "user-123",
  "org_id": "org-456",
  "purpose": "agent-builder",
  "permissions": [
    {
      "resourceType": "org",
      "action": "read"
    },
    {
      "resourceType": "integrations",
      "action": "read"
    },
    {
      "resourceType": "canvases",
      "action": "read",
      "resources": ["canvas-789"]
    }
  ]
}
```

### Minting Rules

- The app must authenticate the caller before minting any scoped token.
- The app must choose a concrete subject before minting:
  - For interactive AI canvas builder requests, the current organization user.
  - For future background jobs, a service account user.
- Before minting, the app must compute the token permissions as the intersection of:
  - what the feature is allowed to request, and
  - what the subject is currently granted by RBAC.
- If the resulting permission set is empty, the app must not mint the token.
- If a request is for a specific canvas, the app must validate the subject can access that canvas
  before minting the token.
- If a request is for a specific canvas, the token must narrow the `canvases` permission through
  `permissions[].resources`.

### Validation Rules

- The API must reject scoped tokens that fail signature validation.
- The API must reject scoped tokens whose `token_type` is not `scoped`.
- The API must reject scoped tokens whose `aud` is not `superplane_api`.
- The API must reject scoped tokens whose `exp` or `nbf` are invalid.
- The API must reject scoped tokens whose subject user cannot be found or is deleted.
- The API must reject scoped tokens whose subject user does not belong to `org_id`.
- Scoped tokens must be validated before falling back to the existing long-lived API token flow.

### Authorization Rules

- Scoped tokens must not replace RBAC.
- Existing RBAC must remain a required enforcement layer for every scoped request.
- Effective authorization must be:
  `scoped permission check AND existing RBAC check`.
- Scoped permissions must never widen the subject's permissions.
- Org-wide permissions are represented by permissions without `resources`.
- Resource-scoped permissions are represented by permissions with `resources`.
- Resource-scoped permissions must fail closed if the target resource cannot be resolved for the
  current RPC.

### gRPC Enforcement Rules

- Scoped permissions must be validated once in the HTTP auth layer when the bearer token is parsed.
- The validated scoped claims must be stored in HTTP request context.
- The gRPC gateway must only propagate the effective scoped permissions into gRPC metadata.
- The gRPC interceptor must not depend on `token_type` or `purpose` metadata.
- The gRPC interceptor must check scoped permissions before RBAC.
- The gRPC interceptor must use explicit per-method resource resolvers for resource-scoped checks.
- Resource resolution must not rely on generic request-shape guessing such as "try `GetId()` or
  `GetCanvasId()` for everything."

### AI Canvas Builder Requirements

- The AI canvas builder must stop depending on `SUPERPLANE_API_TOKEN`.
- The browser must request a chat session from the app, not mint or derive auth on its own.
- The app must expose `Agents.CreateAgentChatSession`.
- `CreateAgentChatSession` must return a scoped JWT for the current organization user.
- The returned token must use `purpose=agent_chat`.
- The returned token must be narrow enough for the current canvas-builder use case.
- The browser may send that scoped JWT to the agent.
- The agent must use the same scoped JWT:
  - to authenticate the browser-to-agent request
  - to authenticate downstream SuperPlane API calls
- The agent must derive `org_id` and `canvas_id` from the scoped JWT, not from the request body.
- The agent must use `SUPERPLANE_BASE_URL` and must not accept a browser-controlled API base URL.

### Background Agent Requirements

- The same scoped token format must support service account subjects later on.
- Background jobs must be able to mint scoped tokens with `purpose` values distinct from
  interactive agent chat.
- A future background agent must not require a new token type to act as a service account.

### Observability and Auditability

- The system must log scoped-token minting with subject, org, purpose, permissions, and expiration.
- The system must not log raw JWT values.
- Scoped-token validation failures must be observable with a reason code.
- API audit trails must continue to attribute actions to the resolved `users.id` subject.

### Backward Compatibility

- Existing browser session cookies must continue to work unchanged.
- Existing long-lived API tokens for human users and service accounts must continue to work.
- The new scoped-token validator must coexist with the current bearer token lookup path.

## Risks & Open Questions

- The rule table in the authorization interceptor must stay in sync with new RPC methods that need
  resource-scoped checks.
- If a new resource-scoped method is added without a resolver, bounded scoped permissions will deny
  the request. This is safer than allowing it, but it is still an operational footgun.
- Using the current JWT secret is acceptable for v1, but future token classes may justify a
  dedicated key.
- `purpose` is currently validated at token-parse time, not enforced as a second-layer policy in
  the gRPC interceptor.
