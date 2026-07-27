# Product Description

## Purpose

SuperPlane is an open-source automation engine for AI-driven engineering. It
orchestrates multi-step workflows across the tools teams already use—Git, LLMs,
CI/CD, observability, incident management, infrastructure, and
communication—with durable execution, approvals, and an operational UI. The
intended outcome is that humans and AI can interact with engineering systems
through explicit, reviewable, and versioned processes rather than ad hoc
scripts. ([README.md:1-7](../../README.md#L1-L7))

## Product Boundaries

### Inside SuperPlane

- Authoring and versioning of app workflows as graphs of triggers and actions.
- Durable run and execution tracking with queues, cancellation, and
  re-emission.
- Human-in-the-loop controls such as approvals, waits, and time gates.
- Canvas-scoped memory for state shared across paths and runs.
- Per-app Console panels for status, KPIs, runbooks, and authorized actions.
- Organization-scoped authentication, RBAC, secrets, integrations, and API keys.
- In-product agent assistance for building and operating canvases, subject to
  feature and permission gates.
- Self-hosted and managed deployment of the SuperPlane engine.

Evidence:
[README.md:21-36](../../README.md#L21-L36),
[docs/contributing/architecture.md:7-27](../../docs/contributing/architecture.md#L7-L27),
[protos/agents.proto:27-94](../../protos/agents.proto#L27-L94).

### Outside SuperPlane

- Replacing the third-party systems being orchestrated.
- Being the system of record for application source code beyond the app's own
  workflow and console files.
- Guaranteeing availability or semantics of external provider APIs.
- Acting as a generic chatbot unrelated to canvas authoring and operation.
- Providing org-global memory shared by every canvas.

Evidence:
[docs/prd/ai-canvas-builder-sidebar.md:32-38](../../docs/prd/ai-canvas-builder-sidebar.md#L32-L38),
[docs/prd/canvas-memory.md:29-33](../../docs/prd/canvas-memory.md#L29-L33).

## Major Capabilities

1. **Apps / canvases:** Deployable units combining a workflow graph, optional
   console, memory, and deterministic execution. Specs are versioned through a
   stage → commit → live promotion loop.
2. **Event-driven orchestration:** Triggers start runs; workers route events,
   execute components, and propagate outputs through channels.
3. **Guardrails:** Approvals, waits, time gates, RBAC, encrypted secrets, and
   scoped API keys constrain who and what may proceed.
4. **Operational Console:** A 12-column panel grid renders live data from
   memory, executions, and runs, and can invoke authorized trigger actions.
5. **Agents and operators:** Per-canvas agent chat, CLI, and skills share the
   same permission model as the UI.
6. **Integrations:** Registry-backed components and triggers for AI, VCS,
   CI/CD, cloud, observability, incident, communication, ticketing, and
   developer-tool providers.

Evidence:
[docs/contributing/architecture.md:94-104](../../docs/contributing/architecture.md#L94-L104),
[docs/prd/console-and-widgets.md:15-23](../../docs/prd/console-and-widgets.md#L15-L23),
[README.md:63-67](../../README.md#L63-L67).

## Core User Journeys

| Journey | Verified path |
| --- | --- |
| Create and publish an app | Create app → edit canvas → stage → commit → live version visible |
| Run and inspect | Manual/event trigger → durable run → sidebar and node inspection |
| Approve gated work | Run waits at approval → authorized approver confirms → downstream continues |
| Operate from Console | Open `?view=console` → view panels → optionally invoke allowed actions |
| Build with agent | Open agent → request change → staging appears → commit to go live |
| Automate via API | Create API key → call REST/gRPC-Gateway endpoints under RBAC |

Evidence:
[web_src/src/App.tsx:114-127](../../web_src/src/App.tsx#L114-L127),
[test/e2e/canvas_staging_commit_publish_test.go:16-32](../../test/e2e/canvas_staging_commit_publish_test.go#L16-L32),
[test/e2e/approvals_test.go:41-48](../../test/e2e/approvals_test.go#L41-L48),
[test/e2e/agent_staging_edit_test.go:27-49](../../test/e2e/agent_staging_edit_test.go#L27-L49).

## Dependencies and Ecosystem

- **Runtime dependencies:** PostgreSQL, RabbitMQ, and the SuperPlane
  application process; production topologies include single-host Compose and
  Kubernetes with external PostgreSQL.
- **External systems:** Provider APIs reached through integrations; credentials
  stored as organization secrets or integration configuration.
- **Clients:** Web UI, CLI, generated OpenAPI SDKs, and optional external coding
  agents using skills.
- **Optional AI services:** Agent HTTP service and model providers behind
  feature and organization policy gates.

Evidence:
[README.md:189-196](../../README.md#L189-L196),
[docs/contributing/architecture.md:17-27](../../docs/contributing/architecture.md#L17-L27),
[docs/prd/inline-config-assistant.md:105-141](../../docs/prd/inline-config-assistant.md#L105-L141).

## Non-Goals

- Fully autonomous execution that mutates or runs workflows without human
  review by default.
- A general-purpose chat product detached from canvas context.
- Org-wide shared memory across unrelated apps.
- Replacing CI systems, incident tools, or infrastructure providers.
- Guaranteeing zero breaking changes while the product remains in beta.

Evidence:
[docs/prd/ai-canvas-builder-sidebar.md:32-38](../../docs/prd/ai-canvas-builder-sidebar.md#L32-L38),
[README.md:19](../../README.md#L19).

## Open Questions

- Where should the product draw the line between workflow orchestration and
  long-running infrastructure provisioning state?
- Which agent modes (builder, operator, architect) become default versus
  experimental?
- Which packaging forms—self-host, cloud, or installable public apps—are
  primary for growth?
- Which Console and memory capabilities are considered core versus advanced?
