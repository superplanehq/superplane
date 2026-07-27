# Runners

## Overview

Runners execute code-oriented workflow components and return results and live
logs to SuperPlane. Builders need predictable runtime selection and
configuration, while operators need task visibility and safe cancellation.

## Terminology

- **Runner:** Execution capacity that performs supported command, script, or
  agent work for a Canvas node.
- **Runner task:** A unit of work assigned to runner capacity.
- **Live log:** Output streamed while a runner task is active.

## Requirements

### REQ-RNER-001: Configure runner work

**User story:** As a workflow builder, I want to select a supported runner
operation and provide its inputs, so that custom code can participate in an
App workflow.

**Acceptance criteria:**

- **AC-RNER-001.1:** When a builder configures a supported runner operation
  with valid inputs, SuperPlane shall accept the node as publishable.
- **AC-RNER-001.2:** When required runner inputs are missing or unsupported,
  SuperPlane shall identify the configuration problem before successful
  execution.

### REQ-RNER-002: Execute and report results

**User story:** As an App operator, I want runner work to report status,
outputs, and failure information, so that downstream workflow behavior is
deterministic.

**Acceptance criteria:**

- **AC-RNER-002.1:** When runner work completes successfully, SuperPlane shall
  expose its output to the configured downstream channel.
- **AC-RNER-002.2:** When runner work fails or times out, SuperPlane shall show
  a failed or cancelled outcome with actionable execution context.

### REQ-RNER-003: Observe live runner logs

**User story:** As an incident responder, I want to view logs while runner work
is active and after it finishes, so that I can diagnose progress and failures.

**Acceptance criteria:**

- **AC-RNER-003.1:** When live logs are available for an active task,
  SuperPlane shall stream newly available output in execution order.
- **AC-RNER-003.2:** When logs cannot be retrieved, SuperPlane shall show an
  unavailable or error state without changing the execution result.

### REQ-RNER-004: Administer runner tasks

**User story:** As an installation administrator, I want to inspect runner
tasks, so that I can identify stuck or resource-intensive work.

**Acceptance criteria:**

- **AC-RNER-004.1:** When an administrator opens runner task administration,
  SuperPlane shall show task identity and current lifecycle state.
- **AC-RNER-004.2:** When a non-administrator requests runner task
  administration, SuperPlane shall reject the request without exposing task
  details.

## Traceability

- **Product context:** [durable execution](../../README.md#how-it-works)
- **Runtime evidence:** [runner components](../../pkg/components/runner/) and
  [runner task administration](../../pkg/public/admin_runner_tasks.go)
- **UI evidence:** [runner node presentation](../../web_src/src/pages/app/mappers/runner.tsx),
  [live log dialog](../../web_src/src/ui/CanvasPage/RunnerLiveLogDialog/), and
  [admin runner tasks](../../web_src/src/pages/admin/RunnerTasks.tsx)
- **Behavior evidence:** [runner logs inspection](../../test/e2e/run_inspector_runner_logs_test.go)
- **Feature blueprint:** [Runner Tasks and Live Logs](../blueprints/features/runner-tasks-and-live-logs.feature.md)

## Open Questions

- How do users select runner fleets, capacity, regions, and isolation levels?
- What limits apply to runtime duration, artifacts, output size, and log
  retention?
