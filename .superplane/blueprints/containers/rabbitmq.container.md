# RabbitMQ Container Blueprint

## Container Summary

RabbitMQ is the asynchronous dispatch and fan-out broker between API actions,
workers, and realtime delivery. SuperPlane uses named exchanges and routing keys
for canvas, event, execution, run, cancellation, email, usage, and agent-session
signals.

Evidence: [`pkg/grpc/actions/messages/`](../../../pkg/grpc/actions/messages/),
[`pkg/workers/`](../../../pkg/workers/), and
[`rabbitmq.yaml`](../../../release/superplane-helm-chart/helm/templates/rabbitmq.yaml).

## Infrastructure

- Deployments may supply `RABBITMQ_URL` for an external broker or enable the
  chart’s single-replica RabbitMQ `StatefulSet` with persistent storage.
- AMQP is the application protocol. The management port is operational and is
  not an application API.
- Consumers use `go-tackle`, stable service/queue names, and reconnect loops.
  Multiple worker replicas compete on the same service queue.
- API replicas running #EventDistributer require fan-out semantics appropriate
  to their service names so each process-local WebSocket hub receives relevant
  events.

## Entry Points and Boundaries

- Publishers serialize generated protobuf messages for workflow lifecycle
  signals; managed-agent stream requests and session events use their concrete
  message contracts.
- #EventRouter consumes `event-created`; #NodeQueueWorker consumes
  `canvas-queue-item-created` and `execution-finished`; #AgentStreamWorker
  consumes `agent-stream-requested`.
- #EventDistributer consumes state events and emits client-facing JSON through
  the API’s WebSocket hub.
- PostgreSQL records referenced by message IDs are authoritative and may be
  loaded during handling.

Related components: [API Gateway and Realtime](../components/api-gateway-and-realtime.component.md),
[Workflow Execution](../components/workflow-execution.component.md), and
[Managed Agents](../components/managed-agents.component.md).

## System Contracts

### Key Contracts

- Delivery is treated as repeatable. Consumers verify current persisted state
  and skip already-routed, finished, missing, or concurrently claimed work.
- A valid message does not carry all canonical state; identifiers locate the
  current PostgreSQL record.
- Consumer failures return errors where redelivery is useful. Connection loops
  retry after broker closure rather than terminating the process.
- RabbitMQ outage degrades latency and realtime updates. `EventRouter` and
  `NodeQueueWorker` include slow PostgreSQL scans as safety nets, but not every
  consumer has an equivalent fallback.
- Publish-after-commit paths can lose a notification if publishing fails; the
  database remains authoritative.

### Integration Boundaries

- Broker retention, dead-lettering, queue durability, and clustering are
  deployment responsibilities and are not fully specified by the local chart.
- Message schema changes must remain coordinated with all producers and
  consumers deployed from the application image.

## Architecture Decision Records

### ADR-001: Use messages as wakeups around durable state

**Context:** Workflow branches and external operations need asynchronous,
independently scalable processing without moving canonical state into a broker.

**Decision:** Persist state in PostgreSQL and publish compact lifecycle messages
through RabbitMQ to trigger consumers and realtime fan-out.

**Consequences:** Workers can scale horizontally and recover by inspecting
state. End-to-end exactly-once delivery is not claimed; idempotent state
transitions and reconciliation remain necessary.

## Open Questions

- Which queues need explicit dead-letter and poison-message policies?
- Which message families lack a database reconciliation path after a failed
  publish?
