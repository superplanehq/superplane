# API Gateway and Realtime Component Blueprint

## Capability Summary

This capability presents one authenticated HTTP surface over native handlers and
proto-defined services, serves the React application, and translates durable
state-change messages into live WebSocket updates.

Evidence: [`pkg/public/server.go`](../../../pkg/public/server.go),
[`pkg/public/ws/`](../../../pkg/public/ws/),
[`pkg/workers/eventdistributer/`](../../../pkg/workers/eventdistributer/),
[`pkg/grpc/services.go`](../../../pkg/grpc/services.go), and
[`protos/`](../../../protos/).

## Core Components

```component
name: PublicHTTPServer
container: API and Web
responsibilities:
  - Routing native HTTP, static assets, callbacks, health, and WebSockets
  - Hosting `grpc-gateway` handlers under `/api/v1`
```

```component
name: GRPCGatewayServices
container: API and Web
responsibilities:
  - Implementing generated service interfaces in process
  - Translating `google.api.http` contracts to typed Go actions
```

```component
name: EventDistributer
container: API and Web
responsibilities:
  - Consuming RabbitMQ lifecycle events
  - Publishing scoped JSON events through `ws.Hub`
```

#PublicHTTPServer depends on #AuthenticationRBAC before invoking
#GRPCGatewayServices. Domain actions persist through model APIs and publish
RabbitMQ messages; #EventDistributer consumes those signals and targets
workflow or agent-session topics. The React application uses the generated
TypeScript client for request/response contracts and WebSockets for invalidation
and incremental state.

## System Contracts

### Key Contracts

- Proto definitions are the source for REST paths and generated clients.
- Gateway JSON rejects unknown request fields and emits unpopulated response
  fields; errors are sanitized before crossing the public boundary.
- `/ws/{workflowId}` requires organization-scoped authentication. Agent session
  subscriptions additionally verify feature enablement and session ownership.
- WebSocket delivery is ephemeral. Consumers must recover through REST state,
  and events must never be treated as the sole record of a transition.
- Per-process `ws.Hub` state means each API replica needs the relevant RabbitMQ
  feed.

### Integration Contracts

- Principal services include `Canvases`, `Organizations`, `Integrations`,
  `Actions`, `Triggers`, `Widgets`, `Secrets`, `Users`, `Groups`, `Roles`,
  `APIKeys`, and `Agents`.
- Canvas messages include `canvas-created`, `canvas-updated`,
  `canvas-staging-updated`, `canvas-deleted`, and `canvas-memory-updated`.
- See [API and Web](../containers/api-and-web.container.md) and
  [RabbitMQ](../containers/rabbitmq.container.md).

## Architecture Decision Records

### ADR-001: Derive public REST contracts from Protocol Buffers

**Context:** Go services, browser clients, CLI clients, and OpenAPI
documentation require aligned contracts.

**Decision:** Define services in `protos/`, run generated handlers in-process,
and generate OpenAPI/client artifacts from the same source.

**Consequences:** Contract drift is reduced, while proto/code generation becomes
part of every API change.

## Open Questions

- How should realtime delivery be tested and operated with multiple API
  replicas and reconnecting clients?
- Should callback/native routes move into a unified route authorization
  inventory?
