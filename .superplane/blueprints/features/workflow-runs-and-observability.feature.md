# Workflow Runs and Observability Feature Blueprint

## Feature Summary

Users can trigger committed workflows, observe runs/events/queue items/node
executions and outputs in realtime, cancel in-flight work, resolve errors, and
re-emit supported trigger events. This feature implements the
[Workflow Triggers](../../requirements/workflow-triggers.md),
[Control Flow and Approvals](../../requirements/control-flow-and-approvals.md),
and [Runs and Operations](../../requirements/runs-and-operations.md)
requirements.

Evidence: [`protos/canvases.proto`](../../../protos/canvases.proto),
[`pkg/workers/`](../../../pkg/workers/), and
[`web_src/src/components/CanvasToolSidebar/`](../../../web_src/src/components/CanvasToolSidebar/).

## Component Blueprint Composition

- [Workflow Execution](../components/workflow-execution.component.md):
  #EventRouter, #NodeQueueWorker, #NodeExecutor, and #RunLifecycle implement the
  durable event-to-run pipeline.
- [API Gateway and Realtime](../components/api-gateway-and-realtime.component.md):
  run/event/execution APIs provide recovery state and #EventDistributer emits
  live transitions.
- [Registry and Runtime](../components/registry-and-runtime.component.md):
  action/trigger contracts receive scoped execution contexts.
- [Authentication and RBAC](../components/authentication-and-rbac.component.md):
  observation requires canvas read; cancellation/hooks require update.

## Feature-Specific Flow

A trigger or manual hook persists a root `CanvasEvent`. Routing against live
edges creates queue items and a `CanvasRun`; node execution emits downstream
events. The frontend combines paginated API history with WebSocket updates.
Cancellation marks run/executions for termination, invokes action cancellation
where supported, and finalizes only after durable work reaches a terminal state.

## System Contracts

- Every execution belongs to the workflow/version/run that produced it.
- Queue and event handlers tolerate repeated notification.
- Independent branches may run concurrently; a node’s FIFO queue preserves its
  own order.
- Run completion must account for terminal events and deleted/consumed queue
  items, not just one execution result.
- Loss of realtime connectivity cannot lose history; REST reads reconstruct it.
- User cancellation is a state transition, not deletion of audit history.

## Requirement Coverage

- **REQ-TRIG-001:** Registered webhook, schedule, and integration triggers emit
  matching events into the durable event router.
- **REQ-TRIG-002:** Trigger-hook invocation validates declared parameters before
  creating the root event and run.
- **REQ-TRIG-003:** Webhook reset operations replace the credential used to
  resolve inbound trigger events.
- **REQ-TRIG-004:** Trigger metadata and custom titles are persisted on runs and
  returned by run history.
- **REQ-FLOW-001:** Named output channels and live graph edges route each emitted
  event only to subscribed downstream nodes.
- **REQ-FLOW-002:** Waiting executions persist state and resume through scheduled
  action calls; cancellation transitions them to terminal state.
- **REQ-FLOW-003:** Time-gate components schedule durable release at an allowed
  window.
- **REQ-FLOW-004:** Approval components expose authorized decision hooks,
  persist execution context, and emit approved or rejected channels.
- **REQ-RUN-001:** Paginated run APIs and realtime invalidation provide current
  and historical run lists.
- **REQ-RUN-002:** Run, event, execution, output, and log APIs reconstruct the
  committed graph and node-level operational detail.
- **REQ-RUN-003:** Run, execution, and queue cancellation stop or remove durable
  work while preserving history.
- **REQ-RUN-004:** Execution actions and error-resolution/re-emission paths
  recover supported failures without deleting the original run.

## Architecture Decision Records

### ADR-001: Combine durable history with ephemeral realtime updates

**Context:** Operators need immediate feedback and reliable post-run inspection.

**Decision:** Persist lifecycle state in PostgreSQL and stream RabbitMQ-derived
WebSocket notifications to refresh the UI.

**Consequences:** History survives disconnects; clients must reconcile duplicate
or missed realtime messages.

## Open Questions

- What run/event retention defaults balance diagnosis with database growth?
- Which cancellation outcomes should distinguish provider cancellation from
  local termination?
