# Organization Agent Settings

## Overview

This PRD defines an Agent Settings section on the General Settings page where organization admins can
enable Agent Mode and configure the OpenAI API key required for it.

The goal is to provide a clear in-app setup flow that combines feature enablement, secure OpenAI API
key configuration, and auditability.

## Problem Statement

Organizations need a clear way to turn on Agent Mode in SuperPlane and complete setup in one place.
Today, there is no dedicated in-app flow that ties enablement and OpenAI key configuration together.
This slows activation, creates confusion in admin setup, and delays Agent Mode adoption.

## Goals

1. Allow authorized organization users to enable Agent Mode for the organization.
2. Allow authorized organization users to configure the OpenAI API key required for Agent Mode.
3. Securely store and protect the key (never expose full raw key after save).
4. Enable key rotation and removal.
5. Provide clear feature status and key validation feedback.
6. Ensure actions are permissioned and auditable.

## Non-Goals

- Managing keys for providers other than OpenAI.
- Per-user personal API keys (this is organization-level only).
- Advanced usage analytics and cost dashboards.
- Fine-grained key scoping beyond OpenAI API-level capabilities.

## Primary Users

- **Organization Admins and Owners**: Configure and maintain the key.
- **Organization Members**: Indirectly benefit from enabled AI features, but cannot manage the key unless permitted.

## User Stories

1. As an organization admin, I can enable Agent Mode for my organization.
2. As an organization admin, I can add an OpenAI API key so Agent Mode can run.
3. As an organization admin, I can verify whether Agent Mode is enabled and whether the key is configured and valid.
4. As an organization admin, I can rotate (replace) the key without downtime confusion.
5. As an organization admin, I can disable Agent Mode for the organization.
6. As a security-conscious admin, I can trust that the platform never shows or logs my full key.

## Functional Requirements

### Settings Page Access

- Add an **Agent Settings** section on **Settings > General**.
- Only authorized roles (Owner and Admin) can view and edit key controls.
- Unauthorized users see read-only status or no section (based on permission policy).

### Section Structure

- The section should follow the same visual structure as existing settings cards (title, description,
  right-aligned toggle, and conditional content area).
- Section title text: **Enable Agent Mode**.
- The top-level action in this section is the toggle.
- Additional configuration controls are shown inside the same section when enabled.

### Feature Enablement

- Provide an explicit toggle: **Enable Agent Mode**.
- When the toggle is ON, show OpenAI API key configuration controls in this section.
- When the toggle is OFF, hide key configuration controls.
- Enabling requires a valid OpenAI API key.
- Disabling prevents Agent Mode from running, even if a key is stored.
- Show clear status for enabled versus disabled states.

### Add and Update Key

- Provide an input field for API key with masked entry.
- "Save key" action persists key securely.
- On save, run lightweight validation (format and optional provider verification call).
- Show success and failure states with actionable error messages.

### Display State

- Show whether a key is configured.
- Never display full key after save.
- Optionally show masked fingerprint (for example last 4 characters) and last updated timestamp.
- Show who last updated the key (if audit data is available in UI).

### Rotate Key

- Reuse update flow to replace current key.
- New key becomes active immediately after successful save.
- Provide clear messaging that replacement overwrites the previous key.

### Remove Key

- "Remove key" action with confirmation modal.
- After removal, if Agent Mode is enabled, it fails gracefully with a clear prompt to reconfigure.

### Validation and Errors

- Handle invalid format.
- Handle failed provider validation (invalid, expired, or revoked key).
- Handle network and timeout failures gracefully.
- Avoid revealing sensitive provider error internals.

### Auditability

- Log key create, update, and delete events with actor, organization, timestamp, and action type.
- Do not log raw key values.

## Security and Compliance Requirements

- Encrypt key at rest using platform-standard secret management.
- Encrypt key in transit with TLS.
- Never expose raw key in UI after submission.
- Never store raw key in logs, analytics, error traces, or client-side storage.
- Restrict retrieval and decryption to server-side runtime paths that require it.
- Apply RBAC checks on all related endpoints.
- Support key deletion semantics consistent with compliance policy (hard delete or secure invalidation).

## UX Requirements

- Section title: **Agent Settings**.
- Helper text explaining why key is needed and who can manage it.
- Primary states:
  - Disabled
  - Enabled, key not configured
  - Enabled, configured (healthy)
  - Enabled, configured (validation issue)
- Form behaviors:
  - Masked input, paste supported.
  - Save button disabled while submitting.
  - Inline validation and non-technical error copy.
- Layout behaviors:
  - Toggle is visible at all times.
  - OpenAI API key input is only visible when toggle is ON.
  - Section uses the same card-style structure as other settings sections on the page.
- Destructive action (Remove key) should be visually separated and require confirmation.

## Architecture Decision

We intentionally keep Agent Mode in a single organization-scoped table for v1:

- `organization_agent_settings` stores both feature state metadata and encrypted OpenAI key material.

This keeps the data model simple while preserving security requirements (encryption at rest, no plaintext
storage, and strict API access controls). We can split metadata and credentials later if needed.

## Database Structure

### Table: organization_agent_settings

One row per organization controls Agent Mode state and stores encrypted key material plus metadata.

| Column | Type | Required | Notes |
|---|---|---|---|
| `id` | `uuid` | yes | Primary key. |
| `organization_id` | `uuid` | yes | FK to `organizations.id`, unique. |
| `agent_mode_enabled` | `boolean` | yes | Default `false`. |
| `openai_api_key_ciphertext` | `bytea` or `text` | no | Encrypted key material only. |
| `openai_key_encryption_key_id` | `varchar(255)` | no | KMS key reference used for encryption. |
| `openai_key_last4` | `varchar(8)` | no | Last 4 chars for UI display only. |
| `openai_key_status` | `varchar(32)` | yes | `not_configured`, `valid`, `invalid`, `unchecked`. |
| `openai_key_validated_at` | `timestamp` | no | Last successful or failed validation time. |
| `openai_key_validation_error` | `text` | no | Sanitized error message, never includes raw key. |
| `updated_by` | `uuid` | no | FK to `users.id` of last updater. |
| `created_at` | `timestamp` | yes | Standard timestamp. |
| `updated_at` | `timestamp` | yes | Standard timestamp. |

### Database Constraints and Indexes

- Unique index on `organization_agent_settings.organization_id`.
- FK constraints for `organization_id` and `updated_by`.
- Check constraint for `openai_key_status` allowed values.
- Never store plaintext OpenAI keys in any column.

## API and Backend Requirements

### Authorization

- Read operations require organization membership with settings read access.
- Write operations require Owner or Admin role.
- All endpoints are organization-scoped and must enforce org boundary checks.

### API Contracts

#### `GET /api/v1/organizations/{organization_id}/agent-settings`

Returns Agent Mode and OpenAI key status for rendering `Settings > General`.

Response `200`:

```json
{
  "organization_id": "uuid",
  "agent_mode_enabled": true,
  "openai_key": {
    "configured": true,
    "last4": "abcd",
    "status": "valid",
    "validated_at": "2026-02-24T12:00:00Z",
    "validation_error": null,
    "updated_at": "2026-02-24T12:00:00Z",
    "updated_by": "uuid"
  }
}
```

#### `PATCH /api/v1/organizations/{organization_id}/agent-settings`

Updates Agent Mode enabled state.

Request body:

```json
{
  "agent_mode_enabled": true
}
```

Rules:

- If `agent_mode_enabled=true` and no valid OpenAI key is configured, return `422`.
- If `agent_mode_enabled=false`, keep stored key by default; do not delete automatically.

Response `200`:

```json
{
  "organization_id": "uuid",
  "agent_mode_enabled": true
}
```

#### `PUT /api/v1/organizations/{organization_id}/agent-settings/openai-key`

Creates or replaces the OpenAI API key.

Request body:

```json
{
  "api_key": "sk-...",
  "validate": true
}
```

Rules:

- `api_key` is write-only and must never be returned.
- If `validate=true`, perform format check plus provider validation call.
- Store encrypted key and metadata in `organization_agent_settings` in one transaction.

Response `200`:

```json
{
  "configured": true,
  "last4": "abcd",
  "status": "valid",
  "validated_at": "2026-02-24T12:00:00Z"
}
```

#### `DELETE /api/v1/organizations/{organization_id}/agent-settings/openai-key`

Deletes configured OpenAI API key and clears metadata.

Response `204` with empty body.

Post-conditions:

- `openai_key_status` is set to `not_configured`.
- `organization_agent_settings.openai_api_key_ciphertext` is cleared.
- If Agent Mode remains enabled, runtime usage should fail with a clear reconfiguration error.

### Error Model

Standardized error payload:

```json
{
  "error": {
    "code": "agent_mode_key_required",
    "message": "Enable Agent Mode requires a valid OpenAI key.",
    "details": {}
  }
}
```

Expected codes:

- `forbidden`
- `organization_not_found`
- `validation_failed`
- `agent_mode_key_required`
- `openai_key_invalid`
- `openai_validation_unavailable`
- `internal_error`

### Audit Events

- `agent_settings.enabled`
- `agent_settings.disabled`
- `agent_settings.openai_key.created`
- `agent_settings.openai_key.updated`
- `agent_settings.openai_key.deleted`

## Acceptance Criteria

1. Admin can enable Agent Mode from **Settings > General**.
2. Admin can successfully save a valid OpenAI key from **Settings > General**.
3. After save, UI shows enabled and configured state without exposing full key.
4. Invalid key attempts show clear error and do not persist invalid secrets.
5. Admin can replace existing key; old key is no longer used.
6. Admin can disable Agent Mode and sees status update to disabled.
7. Unauthorized users cannot enable, disable, create, update, or delete OpenAI settings.
8. Audit events are generated for enable, disable, create, update, and delete actions.
9. No raw key appears in application logs or client-visible payloads.

## Success Metrics

- Percentage of organizations with AI enabled that configure key via self-serve flow.
- Median time-to-first-successful-key-setup.
- Setup failure rate (validation, network, permission).
- Support tickets related to API key onboarding (target reduction).
- Security incidents related to key handling (target: zero).

## Risks and Mitigations

- **Risk:** Users enter wrong key format or non-OpenAI key.  
  **Mitigation:** Inline validation, clear examples, and actionable errors.

- **Risk:** Secret leakage through logs or telemetry.  
  **Mitigation:** Redaction policies, secure logging filters, and tests for sensitive fields.

- **Risk:** Confusion around role permissions.  
  **Mitigation:** Explicit UI messaging for who can manage key and consistent permission errors.

- **Risk:** Downtime during key rotation.  
  **Mitigation:** Atomic replace flow and immediate status feedback.

## Dependencies

- Organization role and permission system.
- Secure secret storage and encryption service.
- OpenAI connectivity for optional validation checks.
- Audit logging infrastructure.
- Existing **Settings > General** page navigation and layout.

## Rollout Plan

- Internal testing with staging organizations.
- Gradual rollout behind feature flag to a subset of organizations.
- Full rollout after monitoring error rate, save success rate, and security checks.

## Open Questions

1. Should non-admin members see configured status, or should visibility be admin-only?
2. Should key validation be required on save, or optional best-effort?
3. Should v1 support multiple organization keys (for example separate environments)?
4. What audit details should be user-visible versus backend-only?
5. Do we need proactive expiry and revocation detection and alerts in v1 or later?
