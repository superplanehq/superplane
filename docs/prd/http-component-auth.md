# HTTP Component Authorization Schemes

## Overview

This PRD defines an authorization configuration model for the HTTP component so users can choose
an auth scheme and securely source credentials from organization secrets.

The goal is to remove manual auth header construction for common patterns (for example,
`Authorization: Bearer ...`) and provide a safer, structured way to configure HTTP auth without
storing raw secrets in workflow node configuration.

## Problem Statement

Today, the HTTP component supports custom headers but has no first-class auth model. Users must
manually build auth headers and often paste credentials directly into configuration fields or
expressions.

This creates several issues:

- No guided auth setup for common schemes.
- Increased risk of secret leakage through misconfiguration.
- Harder-to-read workflows because auth data is hidden inside arbitrary header text.
- Inconsistent team conventions for setting up authenticated requests.

## Goals

1. Add a first-class authorization configuration section to the HTTP component.
2. Support multiple auth schemes in v1 with clear UX and validation.
3. Source all sensitive auth values from organization secrets (secret + key references), not raw plaintext fields.
4. Keep compatibility with existing HTTP component flows that already use manual headers.
5. Preserve execution safety: no raw secret values in node metadata, logs, or emitted output payloads.

## Non-Goals

- Implementing OAuth 2.0 dynamic token exchange flows (authorization code, client credentials token fetch/refresh).
- Auto-refreshing expiring tokens.
- New global secret storage primitives (reuse existing secret/key references).
- Auto-migrating old manual headers into structured auth configuration.

## Primary Users

- **Workflow builders**: Configure authenticated API calls quickly and safely.
- **Platform admins/security-conscious teams**: Enforce secret-based credential handling.

## User Stories

1. As a workflow builder, I can choose an HTTP auth scheme instead of manually crafting auth headers.
2. As a workflow builder, I can select secret keys for credentials so I never paste sensitive values into node config.
3. As a workflow builder, I can configure requests that require no auth, basic auth, bearer token, or API key auth.
4. As a security-conscious admin, I can trust that auth secrets are resolved only at runtime and
   not exposed in logs or UI payloads.
5. As an existing user, my current HTTP nodes continue working without changes.

## Functional Requirements

### Authorization Section

- Add an optional top-level authorization object in HTTP component configuration.
- Add an `authMethod` select field with:
  - `basic`
  - `bearer`
  - `api_key`
- When authorization is not configured, no auth header/query params are auto-generated.
- Existing `headers` and `queryParams` fields remain available for advanced/manual use cases.

### Scheme: Basic Auth

- Fields:
  - `username` (string, required when `basic`)
  - `password` (secret-key reference, required when `basic`)
- Runtime behavior:
  - Resolve password from secrets context.
  - Send `Authorization: Basic <base64(username:password)>`.
- Validation:
  - `username` must be non-empty.
  - `password.secret` and `password.key` must both be provided.

### Scheme: Bearer Token

- Fields:
  - `token` (secret-key reference, required when `bearer`)
  - `prefix` (string, optional, default `Bearer`)
- Runtime behavior:
  - Resolve token from secrets context.
  - Send `Authorization: <prefix> <token>`.
- Validation:
  - `token.secret` and `token.key` must both be provided.
  - `prefix` defaults to `Bearer` if omitted or blank.

### Scheme: API Key

- Fields:
  - `apiKey` (secret-key reference, required when `api_key`)
  - `location` (select, required): `header` or `query`
  - `name` (string, required): header name or query parameter name
- Runtime behavior:
  - Resolve key value from secrets context.
  - If `location=header`, set `<name>: <value>` header.
  - If `location=query`, append `<name>=<value>` query parameter.
- Validation:
  - `apiKey.secret` and `apiKey.key` must both be provided.
  - `name` must be non-empty.

### Precedence and Conflict Handling

- Explicit user-provided `headers` or `queryParams` that set the same auth target should take
  precedence over auto-generated auth values.
  - Example: if `authMethod=bearer` but user already sets `Authorization` header manually, keep the manual header.
- Emit a warning-level log/metadata note for duplicate auth target conflicts (without revealing secret values).

### Setup-Time Validation

- `Setup()` should validate required auth fields according to selected `authMethod`.
- Invalid combinations should return clear, user-facing validation errors.

### Execution-Time Behavior

- Resolve secret values at execution time through existing `core.SecretsContext`.
- If secret resolution fails (missing secret, missing key, permission failure), fail execution
  with a non-sensitive error message.
- Never include resolved secret values in emitted events (`http.request.finished`,
  `http.request.failed`, `http.request.error`) or execution metadata.

## Configuration Shape (Target)

```json
{
  "method": "GET",
  "url": "https://api.example.com/items",
  "authorization": {
    "authMethod": "bearer",
    "token": {
      "secret": "third-party-api",
      "key": "token"
    },
    "prefix": "Bearer"
  }
}
```

```json
{
  "authorization": {
    "authMethod": "api_key",
    "apiKey": {
      "secret": "stripe",
      "key": "api_key"
    },
    "location": "header",
    "name": "X-API-Key"
  }
}
```

## UX Requirements

- Add a new **Authorization** object section in the HTTP component form.
- `authMethod` appears first and controls conditional visibility of child fields.
- Secret-backed fields use the existing `secret-key` selector UX (`secret` + `key`).
- Helper text explains that credentials come from secrets and are resolved at runtime.
- Keep current headers UI unchanged so advanced users can still configure custom headers.

## Backend and Component Requirements

- Update `pkg/components/http/http.go`:
  - Extend `Spec` with `Authorization`.
  - Extend `Configuration()` schema with `authorization` object and conditional fields.
  - Extend `Setup()` validation for auth scheme requirements.
  - Extend request preparation logic in `executeRequest()` to apply auth.
- Reuse the `SecretKeyRef` pattern used by other components (for consistency and type safety).
- Ensure no behavior changes when authorization is absent.

## Security Requirements

- Credentials must be sourced from secrets context only for `basic`, `bearer`, and `api_key` schemes.
- Do not persist resolved secret values in node configuration, metadata, logs, error payloads, or execution outputs.
- Redact sensitive values from internal errors where relevant.
- Maintain existing transport security behavior (TLS handled by destination URL/protocol).

## Backward Compatibility

- Existing HTTP nodes with manual auth headers must continue to execute unchanged.
- New authorization section is additive and optional.
- Default behavior for existing nodes is equivalent to authorization being absent.

## Acceptance Criteria

1. Builder can select one of `basic`, `bearer`, `api_key` in HTTP component configuration.
2. Required fields are enforced per selected auth scheme at setup time.
3. Basic auth sends valid `Authorization: Basic ...` header with password from secret key.
4. Bearer auth sends valid `Authorization` header with token from secret key.
5. API key auth can inject key in either header or query param based on configuration.
6. Missing/invalid secret references fail execution with non-sensitive errors.
7. Existing HTTP nodes without authorization config continue to work unchanged.
8. Resolved credential values never appear in execution outputs, metadata, or logs.

## Success Metrics

- Reduction in support requests related to manual HTTP auth configuration.
- Percentage of new HTTP nodes using structured auth instead of manual auth headers.
- Zero security incidents caused by leaked auth values from this feature.

## Risks and Mitigations

- **Risk:** Users configure both structured auth and conflicting manual headers/query params.  
  **Mitigation:** Define deterministic precedence and emit conflict warnings.

- **Risk:** Secret lookup failures become a frequent runtime failure mode.  
  **Mitigation:** Clear setup-time requirements and actionable runtime errors.

- **Risk:** Future auth schemes require incompatible model changes.  
  **Mitigation:** Keep authorization settings extensible with method-specific fields.

## Rollout Plan

1. Implement backend component changes for auth config + execution behavior.
2. Expose new configuration fields in UI from component definition payload.
3. Add tests for each auth method and conflict scenarios.
4. Roll out to all users (no migration required because feature is additive).

## Open Questions

1. Should API key `name` support expressions, or remain static-only in v1?
2. Should we add `digest` or `oauth2` placeholders now, or defer until implementation is planned?
3. For duplicate auth targets, should we also surface a UI-level warning at design time (not only runtime)?
4. Should `prefix` for bearer support empty string (some APIs use raw token in `Authorization`)?
