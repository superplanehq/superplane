# SuperPlane Feature Requirements

This directory describes the observable product behavior evidenced by the
current SuperPlane repository. It is a product requirements baseline.

## Product Language

An **App** is the user-facing, deployable unit: its workflow, Console, memory,
files, versions, and operational history. A **Canvas** is the graph and runtime
model inside an App. Repository APIs and tests still use `canvas` extensively;
requirements use **App** for product-facing navigation and lifecycle actions,
and **Canvas** only when the graph, canvas-scoped data, or an existing API
contract specifically needs that term.

## Feature Tree

- **Access and tenancy**
  - [Identity and Access](identity-and-access.md)
  - [Organization Administration](organization-administration.md)
- **App lifecycle and authoring**
  - [App Discovery and Lifecycle](app-discovery-and-lifecycle.md)
  - [Canvas Authoring](canvas-authoring.md)
  - [Staging and Versioning](staging-and-versioning.md)
- **Workflow behavior**
  - [Workflow Triggers](workflow-triggers.md)
  - [Control Flow and Approvals](control-flow-and-approvals.md)
  - [Integrations and Secrets](integrations-and-secrets.md)
  - [Cross-App Orchestration](cross-app-orchestration.md)
- **Operations and runtime**
  - [Runs and Operations](runs-and-operations.md)
  - [Runners](runners.md)
  - [Canvas Memory](canvas-memory.md)
- **Operator and assisted experiences**
  - [Console and Widgets](console-and-widgets.md)
  - [AI Canvas Agent](ai-canvas-agent.md)

## Conventions

- Requirement IDs use `REQ-[PREFIX]-NNN`; acceptance criteria use
  `AC-[PREFIX]-NNN.N`.
- Each acceptance criterion is independently observable and testable.
- Requirements state outcomes and policy, not implementation mechanisms.
- A specific persona appears in every user story.
- Each project-specific requirements document links directly to the feature
  blueprint or blueprints that provide its technical path. Each feature
  blueprint links back here to the applicable document and maps individual
  `REQ-*` IDs to implementation paths or explicit gaps.
- Repository-evidence links support the current baseline but do not replace the
  direct requirements-to-blueprint relationship.
- Use [feature-template.md](feature-template.md) only to author a new
  project-specific requirements document.

## Cross-Feature Open Questions

- Which capabilities are guaranteed in self-hosted SuperPlane versus managed
  deployments?
- Which audit events and retention periods are contractual across
  administration, publishing, secrets, AI assistance, and runtime operations?
- Should product-facing API documentation migrate from Canvas terminology to
  App terminology, or retain Canvas as a stable API concept?
