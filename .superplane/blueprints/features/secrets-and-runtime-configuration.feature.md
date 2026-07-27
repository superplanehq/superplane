# Secrets and Runtime Configuration Feature Blueprint

## Feature Summary

Organization users create named secret collections and keys, bind integrations,
and reference sensitive values from component configuration without placing
plaintext in workflow versions. Runtime contexts resolve only the values needed
for an execution. This feature implements the secret-management portion of the
[Integrations and Secrets](../../requirements/integrations-and-secrets.md)
requirements.

Evidence: [`protos/secrets.proto`](../../../protos/secrets.proto),
[`pkg/grpc/secret_service.go`](../../../pkg/grpc/secret_service.go),
[`pkg/workers/contexts/secrets_context.go`](../../../pkg/workers/contexts/secrets_context.go),
[`pkg/models/secret.go`](../../../pkg/models/secret.go), and
[`pkg/configuration/`](../../../pkg/configuration/).

## Component Blueprint Composition

- [Authentication and RBAC](../components/authentication-and-rbac.component.md):
  secret list/create/update/delete and key mutation use organization-scoped
  `secrets` permissions.
- [Registry and Runtime](../components/registry-and-runtime.component.md):
  component schemas describe secret references, while execution contexts resolve
  `SecretKeyRef` values.
- [Integrations and Webhooks](../components/integrations-and-webhooks.component.md):
  integration properties and encrypted credentials flow through a separate
  integration context but share the encryption boundary.
- [Runner Execution](../components/runner-execution.component.md):
  runner environment variables can resolve secret-backed values before task
  submission.

## Feature-Specific Flow

The API validates names and stores encrypted secret values under an
`Organization`. Workflow configuration persists a reference, not the value.
During setup/execution, the context resolves references in tenant scope and
passes plaintext only to the implementation or external service that needs it.
List/describe contracts expose metadata without returning stored key values.

## System Contracts

- Encryption is mandatory at startup unless explicitly disabled with the unsafe
  `NO_ENCRYPTION=yes` mode.
- Secret references resolve within the active organization.
- API reads do not disclose persisted plaintext secret values.
- Renaming/deleting a secret can invalidate committed references; execution must
  fail explicitly rather than silently substituting an empty value.
- Plaintext exists transiently in process memory and may cross an external
  boundary (for example runner environment or integration authentication).
- Configuration validation occurs before runtime where the component contract
  can determine correctness.

## Requirement Coverage

- **REQ-INT-004:** Organization-scoped secret/key CRUD, encrypted persistence,
  metadata-only reads, authorization checks, and runtime reference resolution
  manage sensitive values without disclosing stored plaintext.

## Architecture Decision Records

### ADR-001: Persist references in workflow configuration

**Context:** Version history and Git-backed workflow files must not contain
organization credentials.

**Decision:** Store named secret/key references and resolve values through
runtime contexts.

**Consequences:** Versions remain shareable and auditable without plaintext;
reference lifecycle and authorization become runtime dependencies.

## Open Questions

- What audit record should be emitted for secret reads and external disclosure?
- Should secret rotation support immutable versions to protect in-flight runs?
