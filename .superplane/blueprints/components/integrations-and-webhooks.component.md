# Integrations and Webhooks Component Blueprint

## Capability Summary

This capability binds an organization to an external service, stores encrypted
properties/secrets, resolves integration actions and triggers, synchronizes
provider state, provisions callback subscriptions, and translates inbound
webhooks into trigger behavior.

Evidence: [`pkg/models/integration.go`](../../../pkg/models/integration.go),
[`pkg/models/webhook.go`](../../../pkg/models/webhook.go),
[`pkg/workers/integration_request_worker.go`](../../../pkg/workers/integration_request_worker.go),
[`pkg/workers/webhook_provisioner.go`](../../../pkg/workers/webhook_provisioner.go),
[`pkg/integrations/`](../../../pkg/integrations/), and
[`protos/organizations.proto`](../../../protos/organizations.proto).

## Core Components

```component
name: IntegrationLifecycle
container: API and Web / Workers
responsibilities:
  - Creating and configuring organization `Integration` records
  - Leasing `IntegrationRequest` work for sync and hooks
```

```component
name: WebhookLifecycle
container: API and Web / Workers
responsibilities:
  - Persisting inbound callback endpoints and encrypted metadata
  - Provisioning and cleaning external subscriptions
```

#IntegrationLifecycle uses #RegistryRuntime to resolve `core.Integration` and
setup providers. API writes enqueue durable `IntegrationRequest` records;
workers lease them, perform external calls outside transactions, and persist
the resulting state. #WebhookLifecycle calls the registry’s
`core.WebhookHandler`, then webhook HTTP routes dispatch inbound payloads to the
owning node/integration context.

## System Contracts

### Key Contracts

- Every integration is scoped to an `Organization`; workflow nodes reference a
  validated installation ID when the implementation requires one.
- Sensitive values are encrypted at rest and exposed through integration/secret
  context methods.
- Integration requests use a five-minute lease. A dead worker makes unfinished
  work due again after lease expiry.
- Webhooks transition through `pending`, `provisioning`, and ready/error
  outcomes. Startup resets stuck provisioning records to `pending`.
- External setup/sync calls never hold a long database transaction. Claim and
  finalize transitions use short transactions around the call.
- A provider operation may have succeeded even when local finalization fails;
  handlers and cleanup must tolerate reconciliation/repetition.
- Registry HTTP policy and maximum response size apply to integration calls.

### Integration Contracts

- `SyncContext` includes public base URLs, organization identity, OIDC support,
  HTTP, configuration, and an integration context.
- `WebhookHandlerContext` carries HTTP, integration, webhook, and logger
  capabilities.
- The callback payload limit follows `config.MaxWebhookPayloadSize`.

## Architecture Decision Records

### ADR-001: Separate durable lifecycle state from provider calls

**Context:** Provider APIs are slow and failure-prone and must not retain
database locks.

**Decision:** Claim work transactionally, invoke the provider without a
transaction, and finalize state in a new transaction.

**Consequences:** Database utilization and concurrency improve. The boundary is
not atomic, so provider operations need idempotency or reconciliation.

## Open Questions

- Which integrations provide stable idempotency identifiers for setup and sync?
- What user-visible retry controls should exist for terminal integration or
  webhook errors?
