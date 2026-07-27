# Workers Container Blueprint

## Container Summary

The Workers container runs the same `cmd/server` binary as the API with selected
`START_*` flags and normally without `START_PUBLIC_API`. It owns asynchronous
workflow routing and execution, run lifecycle, integration synchronization,
webhook provisioning, repository provisioning, managed-agent streaming, email,
retention, and cleanup.

Evidence: [`pkg/server/server.go`](../../../pkg/server/server.go),
[`pkg/workers/`](../../../pkg/workers/), and
[`workers.yaml`](../../../release/superplane-helm-chart/helm/templates/workers.yaml).

## Infrastructure

- The Helm `Deployment` enables the principal workers as one process and can be
  replicated with `workers.replicas`.
- Workers require PostgreSQL, RabbitMQ, the registry/runtime definitions,
  encryption and OIDC material, and Git storage. Individual capabilities add
  external integration, agent, email, usage, or runner dependencies.
- Concurrency is bounded inside workers (commonly weighted semaphores), while
  database row locks, leases, or advisory locks prevent duplicate ownership
  across replicas.
- Several workers combine RabbitMQ wakeups with PostgreSQL polling. This lets
  persisted work recover when a notification is lost or the broker is
  temporarily unavailable.

## Entry Points and Boundaries

- RabbitMQ exchanges carry protobuf or JSON messages such as
  `CanvasNodeEventMessage`, `CanvasNodeQueueItemMessage`,
  `CanvasNodeExecutionMessage`, and `AgentStreamRequest`.
- PostgreSQL scans identify pending `CanvasEvent`, ready `CanvasNode`, due
  `IntegrationRequest`, pending `Webhook`, and cleanup candidates.
- #NodeExecutor and runtime contexts invoke registry actions/triggers and may
  call external APIs or #RunnerExecution.
- `RepositoryProvisionerWorker` reacts to `canvas-created` and creates the
  associated repository through `git.Provider`.

Related components: [Workflow Execution](../components/workflow-execution.component.md),
[Integrations and Webhooks](../components/integrations-and-webhooks.component.md),
[Managed Agents](../components/managed-agents.component.md), and
[Runner Execution](../components/runner-execution.component.md).

## System Contracts

### Key Contracts

- Durable work state is committed in PostgreSQL; RabbitMQ messages are dispatch
  signals and state-change notifications.
- Competing workers must claim work before executing it. `EventRouter` locks
  events, `NodeQueueWorker` locks nodes, integration requests use a five-minute
  lease, webhooks transition through `pending`/`provisioning`, and agent streams
  use session locks and heartbeats.
- External HTTP work should not hold database transactions. Integration sync and
  webhook setup explicitly use short claim/finalize transactions around the
  network call.
- Process crashes can leave external side effects after a local rollback.
  Recovery is capability-specific; webhook provisioning resets stuck state and
  agent session creation performs best-effort provider cleanup.
- Worker enablement is configuration-sensitive; a missing required dependency
  can stop startup, while optional managed agents and email consumers remain
  disabled when unconfigured.

### Integration Boundaries

- RabbitMQ redelivery can repeat a notification; consumers must inspect
  PostgreSQL state and treat already-routed/finished/claimed work as a no-op.
- Scaling the whole deployment multiplies every enabled worker class, not just a
  selected workload.

## Architecture Decision Records

### ADR-001: Use one configurable worker binary

**Context:** Worker classes share models, registry definitions, encryption, and
deployment configuration but have different workload profiles.

**Decision:** Select worker goroutines with `START_*` environment variables in
the common server binary.

**Consequences:** Packaging and local operation are simple and Helm can separate
API from workers. Fine-grained scaling requires separate deployments using
different flag sets rather than different binaries.

## Open Questions

- Which worker groups need independent resource limits and autoscaling first?
- Should all persisted-work publishers adopt a transactional outbox rather than
  capability-specific polling recovery?
