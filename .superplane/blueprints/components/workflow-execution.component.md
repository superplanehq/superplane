# Workflow Execution Component Blueprint

## Capability Summary

Workflow execution turns a live canvas graph into durable runs. Triggers create
`CanvasEvent` records; routing creates node queue items; queue workers schedule
executions; executors invoke registry actions; emitted output events continue
the graph until the run finalizer determines a terminal result.

Evidence: [`pkg/workers/event_router.go`](../../../pkg/workers/event_router.go),
[`pkg/workers/node_queue_worker.go`](../../../pkg/workers/node_queue_worker.go),
[`pkg/workers/node_executor.go`](../../../pkg/workers/node_executor.go),
[`pkg/workers/run_initializer.go`](../../../pkg/workers/run_initializer.go), and
[`pkg/models/`](../../../pkg/models/).

## Core Components

```component
name: EventRouter
container: Workers
responsibilities:
  - Claiming pending `CanvasEvent` records
  - Routing events over live edges into `CanvasNodeQueueItem` records
```

```component
name: NodeQueueWorker
container: Workers
responsibilities:
  - Serializing FIFO work per ready `CanvasNode`
  - Creating execution work from claimed queue items
```

```component
name: NodeExecutor
container: Workers
responsibilities:
  - Resolving actions through `Registry`
  - Driving `CanvasNodeExecution` lifecycle and output emission
```

```component
name: RunLifecycle
container: Workers
responsibilities:
  - Initializing, cancelling, terminating, and finalizing `CanvasRun`
```

#EventRouter reads the committed live `CanvasVersion`, creates queue items in a
transaction, then publishes wakeups. #NodeQueueWorker locks ready nodes and
uses #RegistryRuntime to build execution contexts. #NodeExecutor invokes the
resolved action and persists status/output. #RunLifecycle reacts to root,
terminal, queue deletion, and cancellation signals.

## System Contracts

### Key Contracts

- Only the live committed graph routes events; staged files do not execute.
- A root event creates/identifies a run. Output events retain the run through
  their parent execution.
- Event routing and queue creation are transactional. A routed event is not
  routed again on message redelivery.
- A node queue is FIFO by creation time, and node locking prevents concurrent
  processing of the same serial node state.
- Independent graph branches can proceed in parallel.
- RabbitMQ accelerates processing; minute-level pending-event and ready-node
  scans recover missed wakeups.
- Terminal component errors are persisted and exposed; cancellation workers
  propagate stop requests and finalization rather than deleting history.

### Integration Contracts

- Messages use generated canvas protobuf payloads and named routing keys under
  canvas/events/executions exchanges.
- Runtime actions receive explicit `core.ExecutionContext` capabilities instead
  of direct server access.
- See [Registry and Runtime](registry-and-runtime.component.md),
  [RabbitMQ](../containers/rabbitmq.container.md), and
  [PostgreSQL](../containers/postgresql.container.md).

## Architecture Decision Records

### ADR-001: Persist every orchestration stage

**Context:** Workflows span external calls, process restarts, retries, parallel
branches, and user-visible history.

**Decision:** Represent events, queue items, executions, requests, and runs as
durable PostgreSQL state and use RabbitMQ for dispatch.

**Consequences:** Progress is observable and recoverable, but transitions need
idempotency fences and cleanup/retention.

## Open Questions

- Which action classes require stronger idempotency keys for external side
  effects?
- Should queue concurrency and polling intervals become per-workflow or
  per-node configuration?
