# AI Canvas Agent

## Overview

The AI Canvas Agent helps users build, modify, and troubleshoot an App using
natural language while preserving explicit human control. It includes
Canvas-aware chat, reviewable staged edits, component guidance, outcome
tracking, interruption, and narrow field-level suggestions.

## Terminology

- **Agent chat:** A Canvas-aware conversation associated with an App.
- **Proposed edit:** A reviewable change that has not yet become live.
- **Outcome:** A user-defined result used to assess an agent task.

## Requirements

### REQ-AI-001: Canvas-aware assistance

**User story:** As a workflow builder, I want to describe a workflow goal in
natural language, so that SuperPlane can help me select, configure, and connect
appropriate nodes.

**Acceptance criteria:**

- **AC-AI-001.1:** When the builder sends a message in an App, SuperPlane shall
  respond in the context of that App's current Canvas and available
  capabilities.
- **AC-AI-001.2:** When relevant component guidance is unavailable, SuperPlane
  shall continue with qualified guidance rather than falsely claiming that
  specific guidance was applied.

### REQ-AI-002: Review and stage agent edits

**User story:** As a workflow builder, I want agent-generated changes to remain
reviewable before publishing, so that I control their effect on the App.

**Acceptance criteria:**

- **AC-AI-002.1:** When the agent proposes or applies a Canvas edit,
  SuperPlane shall make the resulting staged change visible before it becomes
  live.
- **AC-AI-002.2:** When a proposed change is invalid, SuperPlane shall leave
  the prior Canvas intact and explain what the builder must resolve.

### REQ-AI-003: Control and evaluate agent work

**User story:** As a workflow builder, I want to interrupt an active agent task
and define its intended outcome, so that I can redirect work and judge the
result.

**Acceptance criteria:**

- **AC-AI-003.1:** When the builder interrupts active agent work, SuperPlane
  shall stop further agent activity and retain the conversation state that is
  available for review.
- **AC-AI-003.2:** When the builder defines an outcome, SuperPlane shall
  associate it with the relevant agent chat and make it available in later
  chat retrieval.

### REQ-AI-004: Suggest individual field values safely

**User story:** As a workflow builder, I want a one-shot suggestion for an
eligible configuration field, so that I can author expressions and text faster
without surrendering control of the form.

**Acceptance criteria:**

- **AC-AI-004.1:** When an eligible non-sensitive field is assisted,
  SuperPlane shall present a suggested value that the builder can accept or
  discard.
- **AC-AI-004.2:** When AI features are disabled or the user cannot update the
  Canvas, SuperPlane shall not provide a successful field suggestion.

## Traceability

- **Product context:** [agents and operators](../../README.md#what-it-does)
- **Detailed behavior:** [AI Canvas builder](../../docs/prd/ai-canvas-builder-sidebar.md),
  [component skill awareness](../../docs/prd/ai-agent-component-skill-awareness.md),
  and [inline field assistance](../../docs/prd/inline-config-assistant.md)
- **API evidence:** [agent chat service](../../protos/agents.proto)
- **Behavior evidence:** [agent flow](../../test/e2e/agent_test.go) and
  [agent staging transition](../../test/e2e/agent_staging_edit_test.go)
- **Feature blueprint:** [Managed Canvas Agent](../blueprints/features/managed-canvas-agent.feature.md)

## Open Questions

- What chat retention, sharing, audit, and deletion policies are contractual?
- Which agent actions require separate confirmation beyond normal staging?
