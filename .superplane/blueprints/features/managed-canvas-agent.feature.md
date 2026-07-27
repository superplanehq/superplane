# Managed Canvas Agent Feature Blueprint

## Feature Summary

An authorized user can converse with a managed agent in a canvas, inspect
workflow/runtime context, ask it to stage changes, follow tool progress in
realtime, define an iterative outcome, interrupt work, reset the chat, and then
review/commit proposed edits. This feature is the technical path for the
[AI Canvas Agent](../../requirements/ai-canvas-agent.md) requirements, with
uncovered behavior identified below.

Evidence: [`protos/agents.proto`](../../../protos/agents.proto),
[`pkg/agents/`](../../../pkg/agents/),
[`pkg/workers/agent_stream_worker.go`](../../../pkg/workers/agent_stream_worker.go),
and [`web_src/src/components/AgentSidebar/`](../../../web_src/src/components/AgentSidebar/).

## Component Blueprint Composition

- [Managed Agents](../components/managed-agents.component.md):
  #ManagedAgentService, #AgentStreamWorker, and #AgentToolRegistry own session,
  provider stream, tools, history, heartbeat, and usage.
- [Authentication and RBAC](../components/authentication-and-rbac.component.md):
  the `agents` permission, organization feature gate, and session ownership all
  apply.
- [Git Staging and Versioning](../components/git-staging-and-versioning.component.md):
  agent write tools patch the user’s staging and never publish directly.
- [API Gateway and Realtime](../components/api-gateway-and-realtime.component.md):
  message APIs and ownership-checked WebSocket events drive the sidebar.

## Feature-Specific Flow

Opening chat ensures one user/canvas `AgentSession`. Sending a message updates
provider state, marks the session streaming, and publishes
`agent-stream-requested`. The worker locks and heartbeats the session, persists
provider messages/tool calls, executes allowed tools, tracks usage, and emits
session events. The UI reloads history when needed and presents staged changes
through the normal commit workflow.

## System Contracts

- One user cannot read or subscribe to another user’s agent session.
- Only one stream may mutate a session at a time; follow-up work is rescheduled.
- Reset refuses to replace a streaming session.
- Interrupt makes the local session idle even if provider interruption fails.
- Stuck streams become failed after heartbeat/legacy grace thresholds.
- Agent tools re-check authorization and operate through bounded dependencies.
- Provider loss may recreate a session and replay context; duplicate provider
  events require idempotent persistence identifiers.

## Requirement Coverage

- **REQ-AI-001:** Canvas-scoped sessions, context tools, registry knowledge, and
  provider streaming support Canvas-aware conversation and guidance.
- **REQ-AI-002:** Agent write tools target the user's Git staging area, where
  normal validation and commit review remain required before publication.
- **REQ-AI-003:** Session interruption, persisted history, and outcome APIs let
  users control and later assess agent work.
- **REQ-AI-004:** Gap: current managed-agent chat and staging tools do not
  provide a dedicated one-shot, accept-or-discard field suggestion contract.

## Architecture Decision Records

### ADR-001: Require human commit after agent edits

**Context:** Agent suggestions can alter executable automation.

**Decision:** Give agent tools staging access but no direct live publication.

**Consequences:** Users retain the commit gate and version record; agent success
does not imply the workflow is live.

## Open Questions

- Which high-impact tools should require an explicit per-call approval?
- What user controls should govern conversation retention and provider archive
  deletion?
