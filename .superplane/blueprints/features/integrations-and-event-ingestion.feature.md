# Integrations and Event Ingestion Feature Blueprint

## Feature Summary

Organization administrators connect external services, complete setup,
select capabilities, and use their actions/triggers in canvases. Provider
webhooks and synchronization turn external state into workflow inputs. See the
[Integrations and Secrets](../../requirements/integrations-and-secrets.md) and
[Workflow Triggers](../../requirements/workflow-triggers.md) requirements.

Evidence: [`protos/organizations.proto`](../../../protos/organizations.proto),
[`protos/integrations.proto`](../../../protos/integrations.proto),
[`pkg/integrations/`](../../../pkg/integrations/), and
[`docs/design/integration-setup-flow.md`](../../../docs/design/integration-setup-flow.md).

## Component Blueprint Composition

- [Integrations and Webhooks](../components/integrations-and-webhooks.component.md):
  #IntegrationLifecycle and #WebhookLifecycle own setup, sync, subscriptions,
  callbacks, and cleanup.
- [Registry and Runtime](../components/registry-and-runtime.component.md):
  #RegistryRuntime supplies integration definitions, actions, triggers, setup
  providers, capabilities, and network-limited HTTP.
- [Workflow Execution](../components/workflow-execution.component.md):
  webhook-triggered events enter the normal durable routing pipeline.
- [Authentication and RBAC](../components/authentication-and-rbac.component.md):
  integration CRUD/configuration requires organization-scoped permissions.

## Feature-Specific Flow

API creation persists an `Integration` and due `IntegrationRequest`.
The worker leases and synchronizes it with the external API, persisting ready or
error state. Committing a trigger can create a pending `Webhook`; the provisioner
establishes the external subscription and stores returned metadata. Inbound
callbacks resolve the webhook/node and let the registered handler emit events.
Deletion workers tear down external subscriptions and local records.

## System Contracts

- Integration credentials are encrypted at rest and never returned as plaintext
  API fields.
- An integration ID used by a node must belong to the node’s organization.
- Setup/sync and webhook calls do not hold database transactions.
- Lease expiry and pending-state scans recover worker interruption.
- Callback payload size and outbound HTTP network policy are bounded.
- External success plus local failure is possible; retries must avoid multiplying
  subscriptions or side effects where provider contracts allow.

## Requirement Coverage

- **REQ-INT-001:** Registry definitions and integration APIs expose available
  integrations, capabilities, components, triggers, and resources.
- **REQ-INT-002:** Integration lifecycle records, setup providers, validation,
  and synchronization converge configured connections to ready or error state.
- **REQ-INT-003:** Committed nodes reference an organization-owned integration;
  setup and execution reject missing or cross-tenant bindings.
- **REQ-TRIG-001:** Provisioned provider webhooks resolve their committed trigger
  node and emit inbound payloads into durable workflow routing.
- **REQ-TRIG-003:** Webhook lifecycle APIs and cleanup replace or remove
  externally provisioned callback credentials and metadata.

## Architecture Decision Records

### ADR-001: Bind integrations at organization scope

**Context:** Credentials and reusable provider resources serve multiple
workflows within one tenant.

**Decision:** Store `Integration` under `Organization` and reference it from
committed workflow nodes.

**Consequences:** Connections are reusable while tenant validation and secret
handling are mandatory at every binding.

## Open Questions

- What common idempotency/reconciliation contract should all webhook handlers
  implement?
- Which setup-provider behavior remains development-gated, and what is required
  to make it generally available?
