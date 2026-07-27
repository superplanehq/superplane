# Managed Agents Component Blueprint

## Capability Summary

Managed agents provide a private, per-user canvas conversation whose provider
session can inspect runtime state and modify the user’s staging through a
restricted tool registry. API methods manage sessions/messages/outcomes;
workers stream provider events, execute tools, persist history and usage, and
fan status to the owning user over WebSocket.

Evidence: [`pkg/agents/service.go`](../../../pkg/agents/service.go),
[`pkg/agents/provider.go`](../../../pkg/agents/provider.go),
[`pkg/agents/agent_tools/`](../../../pkg/agents/agent_tools/),
[`pkg/workers/agent_stream_worker.go`](../../../pkg/workers/agent_stream_worker.go),
and [`protos/agents.proto`](../../../protos/agents.proto).

## Core Components

```component
name: ManagedAgentService
container: API and Web
responsibilities:
  - Ensuring/resetting owned `AgentSession` records
  - Sending, interrupting, and defining outcomes through `agents.Provider`
```

```component
name: AgentStreamWorker
container: Workers
responsibilities:
  - Streaming provider events under a session lock
  - Persisting `AgentSessionMessage`, executing custom tools, and publishing updates
```

```component
name: AgentToolRegistry
container: Workers
responsibilities:
  - Exposing authorized canvas read/runtime/staging/integration tools
  - Returning structured tool results without unrestricted server access
```

#ManagedAgentService checks organization/user agent permission and maintains one
session per user/canvas using a PostgreSQL advisory transaction lock.
`agent-stream-requested` wakes #AgentStreamWorker, which limits concurrent
streams, locks the session, heartbeats it, and invokes #AgentToolRegistry.
Session events feed #EventDistributer and the ownership-checked WebSocket topic.

## System Contracts

### Key Contracts

- Sessions and history are private to their creating organization/user.
- The feature and provider must be configured; otherwise agent services are not
  exposed as a functional capability.
- Only one stream owns a session. Locked follow-up turns are durably
  rescheduled, and heartbeats distinguish healthy long turns from stuck state.
- Interrupt marks local state idle first and treats provider interruption as
  best effort; late stream events are dropped by state checks.
- Provider sessions can be recreated; context/tool schema revision tracking
  supports replay after provider loss or tool changes.
- Agent modifications write user staging, never directly mutate the live
  workflow.
- Provider usage is tracked with an idempotency key and published after a run
  completes where configured.

### Integration Contracts

- `agents.Provider` owns create, message/outcome, stream, interrupt, and archive
  operations.
- Agent APIs live under `/api/v1/agents/...`; realtime uses a session-specific
  WebSocket topic.
- See [Authentication and RBAC](authentication-and-rbac.component.md) and
  [Git Staging and Versioning](git-staging-and-versioning.component.md).

## Architecture Decision Records

### ADR-001: Separate request initiation from provider streaming

**Context:** Provider turns are long-running, emit incremental events, and may
invoke server-side tools.

**Decision:** API calls update/enqueue session work; a competing worker consumes
and streams it under a database-backed ownership lock.

**Consequences:** API latency is bounded and stream work scales independently.
Session state, retries, heartbeats, and provider/local reconciliation are
explicit.

## Open Questions

- What retention and deletion policy applies to agent history and provider
  archives?
- Which tool operations require additional approval or audit controls?
