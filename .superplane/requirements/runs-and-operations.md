# Runs and Operations

## Overview

Runs and operations give users evidence that Apps are executing durably and
tools to inspect, stop, retry, and recover work. The operational model spans
run history, per-node executions, queued items, payloads, outcomes, and logs.

## Terminology

- **Run:** One end-to-end workflow instance started by a trigger.
- **Execution:** One node's work within a run.
- **Queue item:** Work waiting for a node's execution capacity.

## Requirements

### REQ-RUN-001: Browse run history

**User story:** As an operator, I want to browse recent and older App runs, so
that I can assess workflow health over time.

**Acceptance criteria:**

- **AC-RUN-001.1:** When runs exist, SuperPlane shall show each run's identity,
  trigger, state or result, and timing information.
- **AC-RUN-001.2:** When history exceeds the initial view, SuperPlane shall
  make older runs reachable without requiring a separate manual reload action.

### REQ-RUN-002: Inspect a run and its nodes

**User story:** As an incident responder, I want to inspect a run's graph and
node details, so that I can locate the source of a failure or delay.

**Acceptance criteria:**

- **AC-RUN-002.1:** When the responder selects a run, SuperPlane shall identify
  the inspected run and show node-level states and outcomes for its version.
- **AC-RUN-002.2:** When the responder opens a node execution, SuperPlane shall
  show available inputs, outputs, result details, and relevant operational
  logs without changing the run.

### REQ-RUN-003: Stop active and queued work

**User story:** As an operator, I want to cancel a run, execution, or queued
item, so that obsolete or unsafe automation stops.

**Acceptance criteria:**

- **AC-RUN-003.1:** When an authorized operator cancels active work,
  SuperPlane shall stop further progress for that work and display a cancelled
  result.
- **AC-RUN-003.2:** When an authorized operator removes a queued item,
  SuperPlane shall remove it from the queue without starting its execution.

### REQ-RUN-004: Recover from failures

**User story:** As an operator, I want supported retry or resolution actions on
failed work, so that I can recover without rerunning unaffected steps.

**Acceptance criteria:**

- **AC-RUN-004.1:** When a failed execution exposes a supported recovery
  action, SuperPlane shall make the action available only to authorized
  operators.
- **AC-RUN-004.2:** When recovery succeeds, SuperPlane shall preserve the
  original run history and show the resulting execution progression.

## Traceability

- **Product context:** [runs and durable execution](../../README.md#how-it-works)
- **API evidence:** [runs, executions, queues, cancellation, and recovery](../../protos/canvases.proto)
- **UI evidence:** [run status](../../web_src/src/ui/Runs/RunStatusBadge.tsx)
  and [runner logs](../../web_src/src/ui/Runs/RunInspectorRunnerLogs.tsx)
- **Behavior evidence:** [runs view](../../test/e2e/runs_view_test.go),
  [queue and execution cancellation](../../test/e2e/canvas_page_test.go), and
  [runner log inspection](../../test/e2e/run_inspector_runner_logs_test.go)
- **Feature blueprint:** [Workflow Runs and Observability](../blueprints/features/workflow-runs-and-observability.feature.md)

## Open Questions

- Which failures are retryable, re-emittable, or resolvable, and how are those
  choices communicated?
- What retention, export, and search guarantees apply to runs and payloads?
